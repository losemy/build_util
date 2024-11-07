package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "build_util",
		Usage: "Put an executable and supplemental files into a zip file that works with Aliyun FunctionCompute.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "",
				Usage:   "output file path for the zip. Defaults to the first input file name.",
			},
		},
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				return errors.New("no input provided")
			}
			c.Args().Slice()

			inputExe := c.Args().First()
			outputZip := c.String("output")
			if outputZip == "" {
				outputZip = fmt.Sprintf("%s.zip", filepath.Base(inputExe))
			}

			if IsDir(inputExe) {
				if err := compressExeAndArgs(outputZip, "", c.Args().Slice()); err != nil {
					return fmt.Errorf("failed to compress file: %v", err)
				}
			} else {
				if err := compressExeAndArgs(outputZip, inputExe, c.Args().Tail()); err != nil {
					return fmt.Errorf("failed to compress file: %v", err)
				}
			}

			log.Print("wrote " + outputZip)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func writeExe(writer *zip.Writer, pathInZip string, data []byte) error {
	if pathInZip != "bootstrap" {
		header := &zip.FileHeader{Name: "bootstrap", Method: zip.Deflate}
		header.SetMode(0755 | os.ModeSymlink)
		link, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := link.Write([]byte(pathInZip)); err != nil {
			return err
		}
	}

	exe, err := writer.CreateHeader(&zip.FileHeader{
		CreatorVersion: 3 << 8,     // indicates Unix
		ExternalAttrs:  0777 << 16, // -rwxrwxrwx file permissions
		Name:           pathInZip,
		Method:         zip.Deflate,
	})
	if err != nil {
		return err
	}

	_, err = exe.Write(data)
	return err
}

func compressExeAndArgs(outZipPath string, exePath string, args []string) error {
	zipFile, err := os.Create(outZipPath)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := zipFile.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to close zip file: %v\n", closeErr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	if len(exePath) != 0 {
		data, err := ioutil.ReadFile(exePath)
		if err != nil {
			return err
		}
		err = writeExe(zipWriter, filepath.Base(exePath), data)
	}

	if err != nil {
		return err
	}

	for _, arg := range args {
		if arg == "scf_bootstrap" {
			bootstrap, err := zipWriter.CreateHeader(&zip.FileHeader{
				CreatorVersion: 3 << 8,     // indicates Unix
				ExternalAttrs:  0777 << 16, // -rwxrwxrwx file permissions
				Name:           arg,
				Method:         zip.Deflate,
			})
			data, err := ioutil.ReadFile(arg)
			if err != nil {
				return err
			}
			_, err = bootstrap.Write(data)
			if err != nil {
				return err
			}
			continue
		}
		if IsDir(arg) {
			//_, err := zipWriter.Create(fileInfo.Name() + "/")
			if err := zipDirectory(zipWriter, arg); err != nil {
				return err
			}
			continue
		}
		writer, err := zipWriter.Create(arg)
		if err != nil {
			return err
		}
		data, err := ioutil.ReadFile(arg)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			return err
		}
	}
	return err
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false // 返回错误，例如路径不存在或无法访问
	}
	return info.IsDir()
}

// Walk through the directory and write the files to the zip file
func zipDirectory(zipWriter *zip.Writer, folderToZip string) error {
	// Get the list of files and directories to zip
	fileInfoArray, err := os.ReadDir(folderToZip)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileInfoArray {
		if fileInfo.IsDir() {
			// Create a new zip file for the directory
			_, err := zipWriter.Create(fileInfo.Name() + "/")
			if err != nil {
				return err
			}
			// Recursively call the function for the subdirectory
			err = zipDirectory(zipWriter, filepath.Join(folderToZip, fileInfo.Name()))
			if err != nil {
				return err
			}
		} else {
			// Open the file to add to the zip
			fileToZip, err := os.Open(filepath.Join(folderToZip, fileInfo.Name()))
			if err != nil {
				return err
			}
			// Create a new zip file for the file
			newZipFile, err := zipWriter.Create(fileInfo.Name())
			if err != nil {
				fileToZip.Close()
				return err
			}
			// Copy the file to the zip
			_, err = io.Copy(newZipFile, fileToZip)
			fileToZip.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
