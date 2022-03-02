package entityDumper

import (
	"bufio"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/uyuni-project/inter-server-sync/schemareader"
)

func DumpAllEntities(options DumperOptions) {
	var outputFolderAbs = options.GetOutputFolderAbsPath()
	validateExportFolder(outputFolderAbs)

	file, err := os.OpenFile(outputFolderAbs+"/sql_statements.sql", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Panic().Err(err).Msg("error creating sql file")
	}

	defer file.Close()
	bufferWriter := bufio.NewWriter(file)
	defer bufferWriter.Flush()

	bufferWriter.WriteString("BEGIN;\n")
	if len(options.ChannelLabels) > 0 {
		db := schemareader.GetDBconnection(options.ServerConfig)
		defer db.Close()
		processAndInsertProducts(db, bufferWriter)
		processAndInsertChannels(db, bufferWriter, options)
	}
	if len(options.ConfigLabels) > 0 {
		db := schemareader.GetDBconnection(options.ServerConfig)
		defer db.Close()
		processConfigs(db, bufferWriter, options)
	}

	if options.OSImages || options.Containers {
		dumpImageData(options)
	}

	bufferWriter.WriteString("COMMIT;\n")
}
