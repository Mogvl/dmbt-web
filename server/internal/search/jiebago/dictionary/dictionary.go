// Package dictionary contains a interface and wraps all io related work.
// It is used by jiebago module to read/write files.
package dictionary

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DictLoader is the interface that could add one token or load
// tokens from channel.
type DictLoader interface {
	Load(<-chan Token)
	AddToken(Token)
}

func loadDictionary(reader io.Reader) (<-chan Token, <-chan error) {
	tokenCh, errCh := make(chan Token), make(chan error)

	go func() {
		defer close(tokenCh)
		defer close(errCh)
		scanner := bufio.NewScanner(reader)
		var token Token
		var line string
		var fields []string
		var err error
		for scanner.Scan() {
			line = scanner.Text()
			fields = strings.Split(line, " ")
			token.text = strings.TrimSpace(strings.Replace(fields[0], "\ufeff", "", 1))
			if length := len(fields); length > 1 {
				token.frequency, err = strconv.ParseFloat(fields[1], 64)
				if err != nil {
					errCh <- err
					return
				}
				if length > 2 {
					token.pos = strings.TrimSpace(fields[2])
				}
			}
			tokenCh <- token
		}

		if err = scanner.Err(); err != nil {
			errCh <- err
		}
	}()
	return tokenCh, errCh

}

// LoadDictionary reads the given file and passes all tokens to a DictLoader.
func LoadDictionary(dl DictLoader, fileName string) error {
	filePath, err := dictPath(fileName)
	if err != nil {
		return err
	}
	dictFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer dictFile.Close()
	tokenCh, errCh := loadDictionary(dictFile)
	dl.Load(tokenCh)

	return <-errCh

}

// LoadDictionaryFromBytes loads dictionary tokens from raw bytes.
// Added for the dmbt-web port: loads dictionary data embedded in the binary.
func LoadDictionaryFromBytes(dl DictLoader, data []byte) error {
	tokenCh, errCh := loadDictionary(strings.NewReader(string(data)))
	dl.Load(tokenCh)

	return <-errCh
}

func dictPath(dictFileName string) (string, error) {
	if filepath.IsAbs(dictFileName) {
		return dictFileName, nil
	}
	var dictFilePath string
	cwd, err := os.Getwd()
	if err != nil {
		return dictFilePath, err
	}
	dictFilePath = filepath.Clean(filepath.Join(cwd, dictFileName))
	return dictFilePath, nil
}
