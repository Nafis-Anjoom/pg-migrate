package internal

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Direction bool

const (
    UP Direction = true
    DOWN Direction = false
)

var (
    fileIsDirectoryError = errors.New("file is a directory")
    fileNotReadableError = errors.New("file is not readable")
    invalidMigrationNameError = errors.New("invalid migration naming")
    directoryNotReadableError = errors.New("directory not readable")
    migrationUnexecutableError = errors.New("unable to execute migration")
    databaseConnectionError = errors.New("unable to connect to database")
    databaseQueryError = errors.New("unable to execute query")
)

const OWNER_READ_MASK fs.FileMode = 0400

type migration struct{
    fileName string
    name string
    version int64
    direction Direction
}

type migrations []*migration

 func (m migrations) Len() int {
    return len(m)
}

func (m migrations) Swap(i, j int) {
    m[i], m[j] = m[j], m[i]
}

func (m migrations) Less(i, j int) bool {
    return m[i].version < m[j].version
}

type migrater struct{
    source string
    database string
    upMigrations migrations
    downMigrations migrations
}


func NewMigrater (source, database string) (*migrater, error) {
    files, err := os.ReadDir(source)
    if err != nil {
        return nil, err
    }

    var upMigrations migrations
    var downMigrations migrations

    for _, file := range files {
        migration, err := parseMigrationEntry(source, file.Name())
        if err != nil {
            return nil, err
        }

        if migration.direction == UP {
            upMigrations = append(upMigrations, migration)
        } else {
            downMigrations = append(downMigrations, migration)
        }
    }

    migrater := &migrater{
        source: source,
        database: database,
        upMigrations: upMigrations,
        downMigrations: downMigrations,
    }

    return migrater, nil
}

func parseMigrationEntry(source, fileName string) (*migration, error) {
    fileSrc := source + "/" + fileName
    fi, err := os.Lstat(fileSrc)
    if err != nil {
        return nil, fmt.Errorf("%w: %s", fileNotReadableError, fileSrc)
    }

    if fi.IsDir() {
        return nil, fmt.Errorf("%w: %s", fileIsDirectoryError, fileSrc)
    }

    if fi.Mode() & OWNER_READ_MASK != OWNER_READ_MASK {
        return nil, fmt.Errorf("%w: %s", fileNotReadableError, fileSrc)
    }

    parts := strings.Split(fileName, ".")
    if len(parts) != 4 {
        return nil, fmt.Errorf("%w: %s. Invalid migration naming scheme", invalidMigrationNameError, fileSrc)
    }

    version, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        return nil, fmt.Errorf("%w: %s. Invalid migration versioning", invalidMigrationNameError, fileSrc)
    }

    name := parts[1]
    var direction Direction
    
    switch parts[2] {
    case "up":
        direction = UP
    case "down":
        direction = DOWN
    default:
        return nil, fmt.Errorf("%w: %s. Invalid stream direction", invalidMigrationNameError, fileSrc)
    }

    if parts[3] != "sql" {
        return nil, fmt.Errorf("%w: %s. Invalid file extension", invalidMigrationNameError, fileSrc)
    }

    migration := &migration {
        fileName: fileName,
        name: name,
        version: version,
        direction: direction,
    }

    return migration, nil
}

func (m *migrater) RunMigrations(direction Direction) error {
    var migrations migrations

    if direction == UP {
        migrations = m.upMigrations
        sort.Sort(migrations)
    } else {
        migrations = m.downMigrations
        sort.Sort(sort.Reverse(migrations))
    }

    conn, err := pgx.Connect(context.Background(), m.database)
    if err != nil {
        log.Println("unable to connect to database")
        return err
    }
    startTime := time.Now()

    defer conn.Close(context.Background())

    tx, err := conn.Begin(context.Background())
    if err != nil {
        log.Println("error starting transaction")
        return err
    }

    log.Println("transaction started")

    for _, migration := range migrations {
        sql, err := os.ReadFile(m.source + "/" + migration.fileName)
        if err != nil {
            log.Println("unable to read file:", migration.fileName, "error:", err)
            return err
        }

        tag, err := tx.Exec(context.Background(), string(sql))
        if err != nil {
            log.Println("unable to execute:", migration.fileName, "error:", err)
            return err
        }
        log.Printf("version: %d, name: %s, operation: %s\n", migration.version, migration.name, tag)
    }

    log.Println("transaction finished")
    tx.Commit(context.Background())
    log.Println("transaction committed")
    
    elapsed := time.Since(startTime)
    log.Printf("Elapsed time: %s\n", elapsed)

    return nil
}

func InitVersionTable(connString string) error {
    conn, err := pgx.Connect(context.Background(), connString)
    if err != nil {
        log.Println("unable to connect to database")
        return err
    }

    defer conn.Close(context.Background())

    query := `CREATE TABLE public.versionTable (version INT NOT NULL);`

    _, err = conn.Exec(context.Background(), query)
    if err != nil {
        log.Println("error initializing version table:", err)
        return err
    }

    return nil
}
