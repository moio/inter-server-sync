package packageDumper

import (
	"database/sql"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/uyuni-project/inter-server-sync/dumper"
	"github.com/uyuni-project/inter-server-sync/schemareader"
	"os"
	"os/exec"
	"path/filepath"
)

var serverDataFolder = "/var/spacewalk"

func DumpPackageFiles(db *sql.DB, schemaMetadata map[string]schemareader.Table, data dumper.DataDumper, outputFolder string) {

	packageKeysData := data.TableData["rhnpackage"]
	table := schemaMetadata[packageKeysData.TableName]
	pathIndex := table.ColumnIndexes["path"]

	exportPoint := 0
	batchSize := 500
	jobs := make(chan fileToCopy, batchSize)
	results := make(chan error, batchSize)
	for w := 1; w <= 5; w++ {
		go worker(w, jobs, results)
	}

	for len(packageKeysData.Keys) > exportPoint {
		upperLimit := exportPoint + batchSize
		if upperLimit > len(packageKeysData.Keys) {
			upperLimit = len(packageKeysData.Keys)
		}
		rows := dumper.GetRowsFromKeys(db, table, packageKeysData.Keys[exportPoint:upperLimit])

		for _, rowPackage := range rows{
			path := rowPackage[pathIndex]
			source := fmt.Sprintf("%s/%s", serverDataFolder, path.Value)
			target := fmt.Sprintf("%s/%s", outputFolder, path.Value)
			jobs <- fileToCopy{source: source, target: target}
		}
		for a := 1; a <= len(rows); a++ {
			error := <-results
			if error != nil{
				log.Fatal().Err(error).Msg("Could not Copy File")
			}
		}
		exportPoint = upperLimit
	}
	close(jobs)
	close(results)
}

type fileToCopy struct {
	source, target string
}

func worker(id int, jobs <-chan fileToCopy, results chan<- error) {
	fmt.Println("worker", id, "started  job")
	for j := range jobs {
		results <- systemCopy(j.source, j.target)
	}
}

func systemCopy(src, dest string) error{
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0770); err != nil {
		return err
	}

	cmd := exec.Command("cp", src, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("error when copy package file: %s -> %s", src, dest))
		return err
	}
	return nil
}
