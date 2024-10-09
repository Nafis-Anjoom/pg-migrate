package main

import (
	// "flag"
	"os"
)

func main() {
    // sourcePtr := flag.String("source", "", "source directory")
    // databasePtr := flag.String("database", "", "database connection url")
    //
    // flag.Parse()

    // migrater := newMigrater(*sourcePtr, *databasePtr)

    folderPath := "./migrations"

    _, err := newMigrater(folderPath, "")
    if err != nil {
        os.Exit(1)
    }
}
