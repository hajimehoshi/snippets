// Copyright 2017 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
)

var (
	datastoreClient *datastore.Client
	developmentMode = false
)

const (
	maxContentSizeInBytes = 10 * 1024
	kindName              = "Snippet"
)

const testForm = `<!DOCTYPE html>
<script>
window.addEventListener('load', _ => {
  document.getElementById('submit-button').addEventListener('click', _ => {
    let content = document.getElementById('content').value;
    fetch('/', {
      method: 'POST',
      body:   content,
    }).then(response => {
      console.log('status:', response.status);
      return response.text();
    }).then(key => {
      console.log('key:', key);
    });
  });
});
</script>
<input id="content" type="text">
<button id="submit-button">Submit</button>
`

type Snippet struct {
	CreatedAt time.Time
	Content   []uint8
}

func getSnippets(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) > 1 {
		keyName := r.URL.Path[1:]
		key := datastore.NameKey(kindName, keyName, nil)
		s := &Snippet{}
		if err := datastoreClient.Get(ctx, key, s); err != nil {
			if err == datastore.ErrNoSuchEntity {
				http.NotFound(w, r)
				return
			}
			msg := fmt.Sprintf("Could not retrieve data: %v", err)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Write(s.Content)
		return
	}

	if developmentMode {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]uint8(testForm))
		return
	}

	http.NotFound(w, r)
}

func postSnippets(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("Could not read the request body: %v", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if len(content) > maxContentSizeInBytes {
		msg := "Request body is too big"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	keyName := fmt.Sprintf("%x", sha256.Sum256(content))
	key := datastore.NameKey(kindName, keyName, nil)

	created := false
	if _, err := datastoreClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		// Search existing one
		s := &Snippet{}
		err := datastoreClient.Get(ctx, key, s)
		if err == nil {
			return nil
		}
		if err != datastore.ErrNoSuchEntity {
			return err
		}

		s = &Snippet{
			CreatedAt: time.Now(),
			Content:   content,
		}
		k := datastore.NameKey(kindName, keyName, nil)
		key, err = datastoreClient.Put(ctx, k, s)
		if err != nil {
			return err
		}
		created = true
		return nil
	}); err != nil {
		msg := fmt.Sprintf("Could not store the request body: %v", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if created {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	io.WriteString(w, keyName)
}

func handleSnippets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		getSnippets(ctx, w, r)
	case http.MethodPost:
		postSnippets(ctx, w, r)
	default:
		s := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(s), s)
	}
}

func main() {
	ctx := context.Background()

	projectID := os.Getenv("GCLOUD_DATASET_ID")

	var err error
	datastoreClient, err = datastore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: It looks like appengine.IsDevAppServer() always returns false.
	// Is there a better way?
	if os.Getenv("DATASTORE_EMULATOR_HOST") != "" {
		developmentMode = true
	}

	http.HandleFunc("/", handleSnippets)
	appengine.Main()
}
