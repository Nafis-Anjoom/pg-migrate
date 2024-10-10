package main

import (
	// "flag"
	"os"
	"pg-migrate/internal"
)

func main() {
    // sourcePtr := flag.String("source", "", "source directory")
    // databasePtr := flag.String("database", "", "database connection url")
    //
    // flag.Parse()

    // migrater := newMigrater(*sourcePtr, *databasePtr)

    source := "./migrations"
    database := "postgres://postgres:admin@localhost:5431/dummy"
    

    _, err := internal.NewMigrater(source, database)
    if err != nil {
        os.Exit(1)
    }

}
