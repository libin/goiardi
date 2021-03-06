/* Authenticate_user functions */

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

package main

import (
	"net/http"
	"encoding/json"
)

type authenticator struct {
	Name, Password string
}
type authRepsonse struct {
	Name string `json:"name"`
	Verified bool `json:"verified"`
}

func authenticate_user_handler(w http.ResponseWriter, r *http.Request){
	/* Suss out what methods to allow */

	dec := json.NewDecoder(r.Body)
	var auth authenticator
	var resp authRepsonse
	if err := dec.Decode(&auth); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
	}

	resp.Name = auth.Name
	resp.Verified = true

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
