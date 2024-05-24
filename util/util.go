package util

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/model"
	"github.com/goccy/go-json"
	"github.com/rabbitmq/amqp091-go"
)

const sigma = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GetWallTimeLimit(limit int64) time.Duration {
	timeWithRedundancy := limit + 100
	return time.Duration(timeWithRedundancy) * time.Millisecond
}

// Generate a token whose length is `n`
func GenToken(n int) (string, error) {
	b := make([]byte, n)
	rng := new(big.Int).SetInt64(int64(len(sigma)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, rng)
		if err != nil {
			ErrorLog(err, "GenToken(): rand.Int")
			return "", err
		}
		b[i] = sigma[idx.Int64()]
	}
	return string(b), nil
}

func Mkdir(parent string) (string, string, error) {
	wdName, err := GenToken(config.FolderNameLen)
	if err != nil {
		return "", "", err
	}
	wdPathName := filepath.Join(parent, wdName)
	err = os.Mkdir(wdPathName, 0755)
	if err != nil {
		ErrorLog(err, "Mkdir(): mkdir")
		return "", "", err
	}
	return wdName, wdPathName, nil
}

func LimitFileReader(filePath string) (*model.OmitString, error) {
	f, err := os.Open(filePath)
	if err != nil {
		ErrorLog(err, "LimitFileReader(): open file")
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		ErrorLog(err, "LimitFileReader(): get file status")
		return nil, errors.New("cannot read file status: " + err.Error())
	}
	allSize := stat.Size()
	if allSize == 0 {
		return nil, nil
	}
	readSize := config.OmitStringLen
	if allSize < readSize {
		readSize = allSize
	}
	buf := make([]byte, readSize)
	bufReader := bufio.NewReader(f)
	n, err := bufReader.Read(buf)
	if err != nil {
		ErrorLog(err, "LimitFileReader(): read file")
		return nil, err
	}
	if int64(n) != readSize {
		err := errors.New("exact read length not equal to the expected")
		ErrorLog(err, "LimitFileReader(): read file")
		return nil, err
	}
	return &model.OmitString{
		S:        string(buf),
		OmitSize: allSize - readSize,
	}, nil
}

func SafeCopy(src string, dst string) error {
	os.Remove(dst)
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		ErrorLog(err, "SafeCopy(): get source file status")
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		err := fmt.Errorf("%s is not a regular file", src)
		ErrorLog(err, "SafeCopy(): source file is not a regular file")
		return err
	}

	source, err := os.Open(src)
	if err != nil {
		ErrorLog(err, "SafeCopy(): open source file")
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		ErrorLog(err, "SafeCopy(): open destination file")
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		ErrorLog(err, "SafeCopy(): copy file")
	}
	return err
}

func MakePublishing(resp model.Response, corId string) amqp091.Publishing {
	bd, err := json.Marshal(resp)
	if err != nil {
		ErrorLog(err, "MakePublishing(): marshal")
		panic(err)
	}
	return amqp091.Publishing{
		ContentType:   "application/json",
		CorrelationId: corId,
		Body:          bd,
	}
}

func InternalError(err error, corId string) amqp091.Publishing {
	resp := model.Response{
		ErrCode: model.IE,
		ErrMsg:  err.Error(),
	}
	return MakePublishing(resp, corId)
}

func CompileError(msg *model.OmitString, corId string) amqp091.Publishing {
	msgStr, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	resp := model.Response{
		ErrCode: model.CE,
		ErrMsg:  "compile error",
		Data:    string(msgStr),
	}
	return MakePublishing(resp, corId)
}

func RunError(err error, res model.ExecResult, corId string) amqp091.Publishing {
	if err == nil {
		err = errors.New("exit code is not zero")
	}
	resStr, merr := json.Marshal(res)
	if merr != nil {
		panic(merr)
	}
	resp := model.Response{
		ErrCode: model.RE,
		ErrMsg:  err.Error(),
		Data:    string(resStr),
	}
	return MakePublishing(resp, corId)
}

func OKResp(resp model.ExecResult, corId string) amqp091.Publishing {
	resStr, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	rep := model.Response{
		ErrCode: model.OK,
		ErrMsg:  "success",
		Data:    string(resStr),
	}
	return MakePublishing(rep, corId)
}

func RunningResp(cas int, corId string) amqp091.Publishing {
	casStr := fmt.Sprintf("%d", cas)
	rep := model.Response{
		ErrCode: model.OK,
		ErrMsg:  "running",
		Data:    casStr,
	}
	return MakePublishing(rep, corId)
}

func WAResp(resp model.ExecResult, corId string) amqp091.Publishing {
	resStr, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	rep := model.Response{
		ErrCode: model.OK,
		ErrMsg:  "wrong answer",
		Data:    string(resStr),
	}
	return MakePublishing(rep, corId)
}
