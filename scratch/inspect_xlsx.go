package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f, err := excelize.OpenFile(`D:\goproject\obj_catalog_fyne_v3\Звіт прийнятих-знятих об’єктів (1).xlsx`)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	// List sheets
	sheets := f.GetSheetList()
	fmt.Printf("Sheets in document: %v\n", sheets)

	for _, sheet := range sheets {
		fmt.Printf("\n--- Sheet: %s ---\n", sheet)
		rows, err := f.GetRows(sheet)
		if err != nil {
			log.Printf("failed to get rows for sheet %s: %v", sheet, err)
			continue
		}
		// Print first 10 rows
		limit := 10
		if len(rows) < limit {
			limit = len(rows)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("Row %d: %q\n", i+1, rows[i])
		}
	}
}
