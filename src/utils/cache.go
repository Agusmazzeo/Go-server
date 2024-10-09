package utils

import (
	"io"
	"os"
)

// SaveResponseToFile is a middleware that saves the response bytes to a file and returns them
func SaveResponseToFile(reader io.ReadCloser, filePath string) ([]byte, error) {
	// Read all the bytes from the response body
	responseBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Save the response bytes to the file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = file.Write(responseBytes)
	if err != nil {
		return nil, err
	}

	// Return the response bytes so the calling function can use them
	return responseBytes, nil
}

// ReadResponseFromFile reads a JSON response from a file and returns the content as a byte slice.
func ReadResponseFromFile(filePath string) ([]byte, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file content
	responseData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return responseData, nil
}
