package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"slashbase.com/backend/internal/dao"
	"slashbase.com/backend/internal/models"
	"slashbase.com/backend/pkg/queryengines"
	"slashbase.com/backend/pkg/queryengines/queryconfig"
)

var cliApp struct {
	CurrentDB *models.DBConnection
}

func handleCmd(cmdText string) {

	if cmdText == "" {
		return
	}
	if cmdText == "exit" {
		os.Exit(1)
		return
	}

	if strings.HasPrefix(cmdText, "\\switch") {
		switchDB(cmdText)
	} else {
		runQuery(cmdText)
	}

}

func switchDB(cmdText string) {
	dbname := strings.Replace(cmdText, "\\switch ", "", 1)

	dbConn, err := dao.DBConnection.GetDBConnectionByName(dbname)
	if err != nil {
		fmt.Printf("no db found by name: '%s'\n", dbname)
		return
	}
	success := queryengines.TestConnection(dbConn, getQueryConfigs(dbConn))
	if !success {
		fmt.Printf("cannot connect to db: '%s'\n", dbname)
		return
	}

	cliApp.CurrentDB = dbConn
	fmt.Printf("connected to: '%s'\n", dbname)
}

func runQuery(queryCmd string) {
	if cliApp.CurrentDB == nil {
		fmt.Printf("no db connected. to connect to existing db run '\\switch db-nick-name'\n")
		return
	}
	result, err := queryengines.RunQuery(cliApp.CurrentDB, queryCmd, getQueryConfigs(cliApp.CurrentDB))
	if err != nil {
		fmt.Printf("error: '%s'\n", err.Error())
		return
	}
	if cliApp.CurrentDB.Type == models.DBTYPE_POSTGRES {
		postgresResult(result)
	} else {
		mongoResult(result)
	}
}

func getQueryConfigs(dbConn *models.DBConnection) *queryconfig.QueryConfig {
	createLog := func(query string) {
		queryLog := models.NewQueryLog(dbConn.ID, query)
		go dao.DBQueryLog.CreateDBQueryLog(queryLog)
	}
	readOnly := false
	return queryconfig.NewQueryConfig(readOnly, createLog)
}

func postgresResult(data map[string]interface{}) {

	if msg, ok := data["message"].(string); ok {
		fmt.Printf("Result: '%s'\n", msg)
		return
	}

	t := table.NewWriter()

	headers := table.Row{}
	for _, colName := range data["columns"].([]string) {
		headers = append(headers, colName)
	}

	allRows := []table.Row{}
	for _, rdata := range data["rows"].([]map[string]interface{}) {
		row := make(table.Row, len(rdata))
		for key, value := range rdata {
			idx, _ := strconv.Atoi(key)
			row[idx] = value
		}
		allRows = append(allRows, row)
	}

	t.SetOutputMirror(os.Stdout)
	defStyle := table.StyleDefault
	defStyle.Format.Header = text.FormatDefault
	t.SetStyle(defStyle)
	t.AppendHeader(headers)
	t.AppendRows(allRows)
	t.Render()
}

func mongoResult(data map[string]interface{}) {

	if msg, ok := data["message"].(string); ok {
		fmt.Printf("Result: '%s'\n", msg)
		return
	}

	allRows := []table.Row{}
	for _, rdata := range data["data"].([]map[string]interface{}) {
		row, _ := json.MarshalIndent(rdata, "", " ")
		allRows = append(allRows, table.Row{string(row)})
	}

	t := table.NewWriter()
	defStyle := table.StyleDefault
	defStyle.Options.SeparateRows = true
	t.SetStyle(defStyle)
	t.SetOutputMirror(os.Stdout)
	t.AppendRows(allRows)
	t.Render()

}
