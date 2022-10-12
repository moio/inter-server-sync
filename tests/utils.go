package tests

import (
	"bufio"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
)

// DataRepository encapsulates I/O operations.
type DataRepository struct {
	DB         *sql.DB
	mock       sqlmock.Sqlmock
	Writer     *bufio.Writer
	mockWriter *MockWriter
}

// CreateDataRepository factory method for the DataRepository
func CreateDataRepository() *DataRepository {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	mock.MatchExpectationsInOrder(true)
	checkErr(err)

	mockWriter := &MockWriter{}
	writerAdapter := bufio.NewWriter(mockWriter)
	return &DataRepository{DB: db, mock: mock, Writer: writerAdapter, mockWriter: mockWriter}
}

// Expect adds data to repository, which can then be retrieved by the tested function.
func (repo *DataRepository) Expect(stm string, columns []string, numRecords int, args ...driver.Value) {

	// simulate table having only one row
	recs := sqlmock.
		NewRows(columns)

	for i := 0; i < numRecords; i++ {
		res := []driver.Value{fmt.Sprintf("%04d", i+1)}
		for j := 1; j < len(columns); j++ {
			res = append(res, fmt.Sprintf("%04d", 1))
		}
		recs = recs.AddRow(res...)
	}
	// add mock expectation
	if len(args) > 0 {
		repo.mock.
			ExpectQuery(stm).
			WithArgs(args...).
			WillReturnRows(recs).
			RowsWillBeClosed()
	} else {
		repo.mock.
			ExpectQuery(stm).
			WillReturnRows(recs).
			RowsWillBeClosed()
	}

}

// ExpectInfoSchema adds data to repository, which can then be retrieved by the tested function.
func (repo *DataRepository) ExpectWithRecords(stm string, recs *sqlmock.Rows, args ...driver.Value) {

	// add mock expectation
	if len(args) > 0 {
		repo.mock.
			ExpectQuery(stm).
			WithArgs(args...).
			WillReturnRows(recs).
			RowsWillBeClosed()
	} else {
		repo.mock.
			ExpectQuery(stm).
			WillReturnRows(recs).
			RowsWillBeClosed()
	}

}

// ExpectationsWereMet checks whether all queued expectations
// were met in order. If any of them was not met - an error is returned.
func (repo *DataRepository) ExpectationsWereMet() error {
	return repo.mock.ExpectationsWereMet()
}

func (repo *DataRepository) GetWriterBuffer() []string {
	err := repo.Writer.Flush()
	checkErr(err)
	return repo.mockWriter.data
}

// MockWriter allows to create a mock bufferWriter object, as it implements the interface
type MockWriter struct {
	data []string
}

func (mr *MockWriter) Write(p []byte) (n int, err error) {

	mr.data = append(mr.data, string(p))
	return len(p), nil
}

func (mr *MockWriter) GetData() []string {
	return mr.data
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
