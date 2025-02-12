package polars

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -L${SRCDIR}/lib -lpolars_go
#cgo linux LDFLAGS: -Wl,-rpath=${SRCDIR}/lib
#cgo darwin LDFLAGS: -Wl,-rpath,${SRCDIR}/lib
#include "polars_go.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unsafe"
)

func init() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Failed to get executable path:", err)
		return
	}
	libPath := filepath.Join(filepath.Dir(exePath), "polars/lib")

	err = os.Setenv("LD_LIBRARY_PATH", libPath+":"+os.Getenv("LD_LIBRARY_PATH"))
	if err != nil {
		fmt.Println("Failed to set LD_LIBRARY_PATH:", err)
	}
}

type DataFrame struct {
	ptr *C.CDataFrame
}

type Expr struct {
	ptr *C.CExpr
}

func (df *DataFrame) String() string {
	if df.ptr == nil || df.ptr.handle == nil {
		return "<nil DataFrame>"
	}

	cStr := C.print_dataframe(df.ptr)
	if cStr == nil {
		return "<error printing DataFrame>"
	}
	defer C.free(unsafe.Pointer(cStr))

	return C.GoString(cStr)
}

func ReadCSV(filePath string) (*DataFrame, error) {
	cPath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cPath))

	df := C.read_csv(cPath)
	if df == nil || (*C.CDataFrame)(df).handle == nil {
		return nil, errors.New(C.GoString(C.get_last_error()))
	}

	return &DataFrame{ptr: (*C.CDataFrame)(df)}, nil
}

func (df *DataFrame) Free() {
	if df.ptr != nil {
		C.free_dataframe(df.ptr)
		df.ptr = nil
	}
}

func (df *DataFrame) Width() int {
	return int(C.dataframe_width(df.ptr))
}

func (df *DataFrame) Height() int {
	return int(C.dataframe_height(df.ptr))
}

func (df *DataFrame) Columns() []string {
	var names []string
	for i := 0; ; i++ {
		cStr := C.dataframe_column_name(df.ptr, C.size_t(i))
		if cStr == nil {
			break
		}
		defer C.free(unsafe.Pointer(cStr))
		names = append(names, C.GoString(cStr))
	}
	return names
}

func (df *DataFrame) Filter(expr Expr) *DataFrame {
	filteredPtr := C.filter(df.ptr, expr.ptr)
	if filteredPtr == nil {
		err := errors.New(C.GoString(C.get_last_error()))
		log.Printf("Error while filtering: %s", err)
		return &DataFrame{}
	}
	return &DataFrame{ptr: (*C.CDataFrame)(filteredPtr)}
}

func Col(name string) Expr {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return Expr{ptr: (*C.CExpr)(C.col(cName))}
}

func (e Expr) Gt(value int64) Expr {
	return Expr{ptr: (*C.CExpr)(C.col_gt(e.ptr, C.long(value)))}
}

func (df DataFrame) WriteCSV(filePath string) error {
	cFilePath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cFilePath))

	res := C.write_csv(df.ptr, cFilePath)

	if res == nil {
		return errors.New("write_csv error: unknown failure")
	}

	msg := C.GoString(res)
	if msg != "CSV written successfully" {
		return fmt.Errorf("write_csv error: %s", msg)
	}
	return nil
}

func (df DataFrame) Head(n int) *DataFrame {
	cHeadDf := C.head(df.ptr, C.size_t(n))

	if cHeadDf == nil || (*C.CDataFrame)(cHeadDf).handle == nil {
		err := C.GoString(C.get_last_error())
		log.Printf("Error getting head: %s", err)
		return &DataFrame{}
	}

	return &DataFrame{ptr: (*C.CDataFrame)(cHeadDf)}
}
