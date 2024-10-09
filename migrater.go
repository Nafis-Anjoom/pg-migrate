package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type direction bool

func (d direction) String() string {
    if d {
        return "UP"
    }
    return "DOWN"
}

var (
    UP direction = true
    DOWN direction = false
)

var OWNER_READ_MASK fs.FileMode = 0400

type migration struct{
    fileName string
    version int64
    direction direction
    sql string
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


func newMigrater (source, database string) (*migrater, error) {
    files, err := os.ReadDir(source)
    if err != nil {
        log.Println("error reading directory:", source)
        return nil, err
    }

    // TODO: check database connection

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
        log.Printf("%+v\n", migration)
    }

    migrater := &migrater{
        source: source,
        database: database,
        upMigrations: upMigrations,
    }

    return migrater, nil
}

func parseMigrationEntry(source, fileName string) (*migration, error) {
    // check each file
    fi, err := os.Lstat(source + "/" + fileName)
    if err != nil {
        log.Println(err)
        return nil, err
    }

    if fi.IsDir() {
        log.Println("file is a directory:", fileName)
        return nil, err
    }

    if fi.Mode() & OWNER_READ_MASK != OWNER_READ_MASK {
        log.Println("file not readable:", fileName)
        return nil, err
    }

    parts := strings.Split(fileName, ".")
    if len(parts) != 4 {
        log.Println("invalid migration file:", fileName)
        return nil, err
    }

    version, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        log.Println("invalid version:", fileName)
    }

    var direction direction
    
    switch parts[2] {
    case "up":
        direction = UP
    case "down":
        direction = DOWN
    default:
        log.Println("invalid migration direction:", fileName)
        return nil, err
    }

    if parts[3] != "sql" {
        log.Println("invalid file type:", fileName)
        return nil, err
    }

    migration := &migration {
        fileName: fileName,
        version: version,
        direction: direction,
    }

    return migration, nil
}

func (m *migrater) runMigrations(direction direction) error {
    var migrations migrations

    if direction == UP {
        migrations = m.upMigrations
    } else {
        migrations = m.downMigrations
    }
    sort.Sort(migrations)

    conn, err := pgx.Connect(context.Background(), m.database)
    if err != nil {
        log.Println("unable to connect to database")
        return err
    }

    defer conn.Close(context.Background())


    batch := &pgx.Batch{}
    tx, err := conn.Begin(context.Background())
    if err != nil {
        log.Println("error starting transaction")
        return err
    }

    log.Println("transaction started")

    for _, migration := range migrations {
        sql, err := os.ReadFile(m.source + migration.fileName)
        if err != nil {
            log.Println("unable to read file:", migration.fileName)
        }

        batch.Queue(string(sql))
        log.Println("queued:", migration.fileName)
    }

    br := tx.SendBatch(context.Background(), batch)
    // if err != nil {
    //     log.Println(err)
    //     return err
    // }
    log.Println("transaction finished")

    tx.Commit(context.Background())
    log.Println("transaction committed")

    br.Close()
    return nil
}
