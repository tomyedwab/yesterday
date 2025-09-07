package applications

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/tomyedwab/yesterday/applib/httputils"
	"github.com/tomyedwab/yesterday/nexushub/packages"
)

func HandleInstall(w http.ResponseWriter, r *http.Request) {
	packageManager := packages.NewPackageManager()
	packageName := uuid.New().String()

	// Create a new file with the generated filename
	filename := fmt.Sprintf("%s/%s.zip", packageManager.GetPkgDir(), packageName)
	file, err := os.Create(filename)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to create file: %v", err), http.StatusInternalServerError)
		return
	}
	// TODO(tom) Delete the file before returning
	defer file.Close()

	// Parse the FormData in Body and write to output file
	formData, err := r.MultipartReader()
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to read request body: %v", err), http.StatusInternalServerError)
		return
	}

	for {
		part, err := formData.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to read form data: %v", err), http.StatusInternalServerError)
			return
		}

		_, err = io.Copy(file, part)
		if err != nil {
			httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to write to file: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Create a 6-byte sequence from the current timestamp and two random bytes
	// Then base64-encode the sequence to derive a new instance ID
	seq := make([]byte, 6)
	binary.BigEndian.PutUint32(seq[:4], uint32(time.Now().Unix()))
	rand.Read(seq[4:])
	instanceID := base64.StdEncoding.EncodeToString(seq)

	err = packageManager.InstallPackage(packageName, instanceID)
	if err != nil {
		httputils.HandleAPIResponse(w, r, nil, fmt.Errorf("failed to install package: %v", err), http.StatusInternalServerError)
		return
	}

	httputils.HandleAPIResponse(w, r, map[string]string{
		"instanceId": instanceID,
	}, nil, http.StatusOK)
}
