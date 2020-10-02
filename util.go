package report_receiver_bot

import (
    "os"
    "io"
    "net/http"
    "io/ioutil"
)


func ReadFileBytes(fromPath string) ([]byte, error) {
    file, err := os.Open(fromPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
	return ioutil.ReadAll(file)
}


func DownloadFile(fromURL string, toPath string) error {
    resp, err := http.Get(fromURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    out, err := os.Create(toPath)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}
