/* Sandbox structs, for testing whether cookbook files need to be uploaded */

/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sandbox

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/util"
	"fmt"
	"crypto/md5"
	"crypto/rand"
	"io"
	"log"
	"time"
)

/* The structure of the sandbox responses is... inconsistent. */
type Sandbox struct {
	Id string
	CreationTime time.Time
	Completed bool
	Checksums []string
}

/* We actually generate the sandbox_id ourselves, so we don't pass that in. */
func New(checksum_hash map[string]interface{}) (*Sandbox, error){
	/* For some reason the checksums come in a JSON hash that looks like
	 * this:
 	 * { "checksums": {
	 * "385ea5490c86570c7de71070bce9384a":null,
  	 * "f6f73175e979bd90af6184ec277f760c":null,
  	 * "2e03dd7e5b2e6c8eab1cf41ac61396d5":null
  	 * } } --- per the chef server api docs. Not sure why it comes in that
	 * way rather than as an array, since those nulls are apparently never
	 * anything but nulls. */

	/* First generate an id for this sandbox. Collisions are certainly
	 * possible, so we'll give it five tries to make a unique one before
	 * bailing. This may later turn out not to be the ideal sandbox creation
	 * method, but we'll see. */
	var sandbox_id string
	var err error
	ds := data_store.New()
	for i := 0; i < 5; i++ {
		sandbox_id, err = generate_sandbox_id()
		if err != nil {
			/* Something went very wrong. */
			return nil, err 
		}
		if _, found := ds.Get("sandbox", sandbox_id); found {
			err = fmt.Errorf("Collision! Somehow %s already existed as a sandbox id on attempt %d. Trying again.", sandbox_id, i)
			sandbox_id = ""
			log.Println(err)
		}
	}

	if sandbox_id == "" {
		err = fmt.Errorf("Somehow every attempt to create a unique sandbox id failed. Bailing.")
		return nil, err
	} 
	checksums := make([]string, len(checksum_hash))
	j := 0
	for k, _ := range checksum_hash {
		checksums[j] = k
		j++
	}

	sbox := &Sandbox{
		Id: sandbox_id,
		CreationTime: time.Now(),
		Completed: false,
		Checksums: checksums,
	}
	return sbox, nil
}

func generate_sandbox_id() (string, error) {
	randnum := 20
	b := make([]byte, randnum)
	n, err := io.ReadFull(rand.Reader, b)
	if n != len(b) || err != nil {
		return "", err
	}
	id_md5 := md5.New()
	id_md5.Write(b)
	sandbox_id := fmt.Sprintf("%x", id_md5.Sum(nil))
	return sandbox_id, nil
}

func Get(sandbox_id string) (*Sandbox, error){
	ds := data_store.New()
	sandbox, found := ds.Get("sandbox", sandbox_id)
	if !found {
		err := fmt.Errorf("Sandbox %s not found", sandbox_id)
		return nil, err
	}
	return sandbox.(*Sandbox), nil
}

func (s *Sandbox) Save() error {
	ds := data_store.New()
	ds.Set("sandbox", s.Id, s)
	return nil
}

func (s *Sandbox) Delete() error {
	ds := data_store.New()
	ds.Delete("sandbox", s.Id)
	return nil
}

func GetList() []string {
	ds := data_store.New()
	sandbox_list := ds.GetList("sandbox")
	return sandbox_list
}

func (s *Sandbox) UploadChkList() map[string]map[string]interface{} {
	/* Uh... */
	chksum_stats := make(map[string]map[string]interface{})
	for _, chk := range s.Checksums {
		chksum_stats[chk] = make(map[string]interface{})
		k, _ := filestore.Get(chk)
		if k != nil {
			chksum_stats[chk]["needs_upload"] = false
		} else {
			item_url := fmt.Sprintf("/file_store/%s", chk)
			chksum_stats[chk]["url"] = util.CustomURL(item_url)
			chksum_stats[chk]["needs_upload"] = true
		}

	}
	return chksum_stats
}

func (s *Sandbox) IsComplete() error {
	for _, chk := range s.Checksums {
		k, _ := filestore.Get(chk)
		if k == nil {
			err := fmt.Errorf("Checksum %s not uploaded yet, %s not complete, cannot commit yet.", chk, s.Id)
			return err
		}
	}
	return nil
}

func (s *Sandbox) GetName() string {
	return s.Id
}

func (s *Sandbox) URLType() string {
	return "sandboxes"
}
