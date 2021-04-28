package dumper

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/uyuni-project/inter-server-sync/schemareader"
	"github.com/uyuni-project/inter-server-sync/sqlUtil"
	"strings"
)

func DumpAllTablesData(db *sql.DB, writter *bufio.Writer, schemaMetadata map[string]schemareader.Table,
	startingTables []schemareader.Table, whereFilterClause func(table schemareader.Table) string, onlyIfParentExistsTables []string) {

	processedTables := make(map[string]bool)
	// exporting from the starting tables.
	for _, startingTable := range startingTables{
		processedTables = printAllTableData(db, writter, schemaMetadata, startingTable, whereFilterClause, processedTables, make([]string, 0), onlyIfParentExistsTables)
	}
	// Export tables not exported when exporting the starting tables
	for schemaTableName, schemaTable := range schemaMetadata{
		if !schemaTable.Export{
			continue
		}
		_, ok := processedTables[schemaTableName]
		if ok {
			continue
		}
		exportAllTableData(db, writter, schemaMetadata, schemaTable, whereFilterClause, onlyIfParentExistsTables)
	}
}

func printAllTableData(db *sql.DB, writter *bufio.Writer, schemaMetadata map[string]schemareader.Table, table schemareader.Table,
	whereFilterClause func(table schemareader.Table) string, processedTables map[string]bool, path []string, onlyIfParentExistsTables[]string) map[string]bool {
	log.Debug().Msgf("%s", "Processing table: " + table.Name)
	_, tableProcessed := processedTables[table.Name]
	currentTable := schemaMetadata[table.Name]
	if tableProcessed || !currentTable.Export {
		return processedTables
	}
	path = append(path, table.Name)
	processedTables[table.Name] = true

	for _, reference := range table.References {
		tableReference, ok := schemaMetadata[reference.TableName]
		if !ok || !tableReference.Export{
			continue
		}
		log.Debug().Msgf("%s", "Table processed: " + table.Name)
		printAllTableData(db, writter, schemaMetadata, tableReference, whereFilterClause, processedTables, path, onlyIfParentExistsTables)

	}

	exportAllTableData(db, writter, schemaMetadata, table, whereFilterClause, onlyIfParentExistsTables)

	for _, reference := range table.ReferencedBy {
		tableReference, ok := schemaMetadata[reference.TableName]
		if !ok || !tableReference.Export{
			continue
		}
		if !shouldFollowReferenceToLink(path, table, tableReference) {
			continue
		}
		printAllTableData(db, writter, schemaMetadata, tableReference, whereFilterClause, processedTables, path, onlyIfParentExistsTables)

	}
	return processedTables
}

func exportAllTableData(db *sql.DB, writter *bufio.Writer, schemaMetadata map[string]schemareader.Table, table schemareader.Table,
	whereFilterClause func(table schemareader.Table) string, onlyIfParentExistsTables []string) {

	formattedColumns := strings.Join(table.Columns, ", ")
	sql := fmt.Sprintf(`SELECT %s FROM %s %s;`, formattedColumns, table.Name, whereFilterClause(table))
	rows := sqlUtil.ExecuteQueryWithResults(db, sql)

	for _, row := range rows {
		writter.WriteString(generateRowInsertStatement(db, row, table, schemaMetadata, onlyIfParentExistsTables) + "\n")
	}

}