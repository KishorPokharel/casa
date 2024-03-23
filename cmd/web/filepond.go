package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
)

func (app *application) handleFileUpload(name string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(5 << 20) // 5MB
		if err != nil {
			app.logger.Error(err.Error())
			app.clientError(w, http.StatusBadRequest)
			return
		}
		uploadedFile, _, err := r.FormFile(name)
		if err != nil {
			app.logger.Error(err.Error())
			app.clientError(w, http.StatusBadRequest)
			return
		}
		defer uploadedFile.Close()

		b, err := io.ReadAll(uploadedFile)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		mtype := mimetype.Detect(b)

		allowedExtensions := []string{".jpg", ".jpeg", ".png"}

		ext := mtype.Extension()
		if !slices.Contains(allowedExtensions, ext) {
			app.logger.Warn("invalid image")
			app.serverError(w, r, err)
			return
		}

		name := fmt.Sprintf("%s%s", uuid.NewString(), ext)
		out, err := os.Create(filepath.Join(tmpDir, name))
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		defer out.Close()

		_, err = out.Write(b)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		fmt.Fprint(w, name)
	}
}
