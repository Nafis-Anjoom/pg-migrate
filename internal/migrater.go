package internal

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type direction bool

var (
    UP direction = true
    DOWN direction = false
)

var (
    fileDirectoryError = errors.New("file is a directory")
    fileNotReadableError = errors.New("file is not readable")
    invalidMigrationNameError = errors.New("invalid migration naming")
)

var OWNER_READ_MASK fs.FileMode = 0400

type migration struct{
    fileName string
    name string
    version int64
    direction direction
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
        log.Println("error reading directory:", source)
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
    fi, err := os.Lstat(source + "/" + fileName)
    if err != nil {
        log.Println(err)
        return nil, err
    }

    if fi.IsDir() {
        log.Println("file is a directory:", fileName)
        return nil, fileDirectoryError
    }

    if fi.Mode() & OWNER_READ_MASK != OWNER_READ_MASK {
        log.Println("file not readable:", fileName)
        return nil, fileNotReadableError
    }

    parts := strings.Split(fileName, ".")
    if len(parts) != 4 {
        log.Println("invalid migration file:", fileName)
        return nil, invalidMigrationNameError
    }

    version, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        log.Println("invalid version:", fileName)
        return nil, invalidMigrationNameError
    }

    name := parts[1]
    var direction direction
    
    switch parts[2] {
    case "up":
        direction = UP
    case "down":
        direction = DOWN
    default:
        log.Println("invalid migration direction:", fileName)
        return nil, invalidMigrationNameError
    }

    if parts[3] != "sql" {
        log.Println("invalid file type:", fileName)
        return nil,invalidMigrationNameError 
    }

    migration := &migration {
        fileName: fileName,
        name: name,
        version: version,
        direction: direction,
    }

    return migration, nil
}

func (m *migrater) RunMigrations(direction direction) error {
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
        // log.Printf("%s: %s\n", tag, migration.name)
    }

    log.Println("transaction finished")
    tx.Commit(context.Background())
    log.Println("transaction committed")
    
    elapsed := time.Since(startTime)
    log.Printf("Elapsed time: %s\n", elapsed)

    return nil
}
