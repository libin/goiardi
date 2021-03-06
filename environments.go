/* Environment functions */

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
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"fmt"
	"encoding/json"
)

func environment_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	path_array := SplitPath(r.URL.Path)
	env_response := make(map[string]interface{})
	num_results := r.FormValue("num_versions")

	path_array_len := len(path_array)

	if path_array_len == 1 {
		switch r.Method {
			case "GET":
				env_list := environment.GetList()
				for _, env := range env_list {
					item_url := fmt.Sprintf("/environments/%s", env)
					env_response[env] = util.CustomURL(item_url)
				}
			case "POST":
				env_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				}
				chef_env, err := environment.Get(env_data["name"].(string))
				if chef_env != nil {
					httperr := fmt.Errorf("Environment %s already exists.", env_data["name"].(string))
					JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
					return
				}
				chef_env, err = environment.NewFromJson(env_data)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := chef_env.Save(); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				}
				env_response["uri"] = util.ObjURL(chef_env)
				w.WriteHeader(http.StatusCreated)
			default:
				JsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
				return
		}
	} else if path_array_len == 2 {
		/* All of the 2 element operations return the environment
		 * object, so we do the json encoding in this block and return 
		 * out. */
		env_name := path_array[1]
		env, err := environment.Get(env_name)
		del_env := false /* Set this to delete the environment after
				  * sending the json. */
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		switch r.Method {
			case "GET", "DELETE":
				/* We don't actually have to do much here. */
				if r.Method == "DELETE" {
					del_env = true
				}
			case "PUT":
				env_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				}
				if env_name != env_data["name"].(string) {
					env, err = environment.Get(env_data["name"].(string))
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusConflict)
						return
					} else {
						env, err = environment.NewFromJson(env_data)
						if err != nil {
							JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
							return
						}
					}
				} else {
					if err := env.UpdateFromJson(env_data); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					}
				}
				if err := env.Save(); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				}
			default:
				JsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
				return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(&env); err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
		if del_env {
			err = env.Delete()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
		}
		return
	} else if path_array_len == 3 {
		env_name := path_array[1]
		op := path_array[2]

		if op == "cookbook_versions" && r.Method != "POST" || op != "cookbook_versions" && r.Method != "GET" {
			JsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}

		env, err := environment.Get(env_name)
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		switch op {
			case "cookbook_versions":
				/* Chef Server API docs aren't even remotely
				 * right here. What it actually wants is the
				 * usual hash of info for the latest or
				 * constrained version. Weird. */
				cb_ver, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				}

				if _, ok := cb_ver["run_list"]; !ok {
					JsonErrorReport(w, r, "POSTed JSON badly formed.", http.StatusMethodNotAllowed)
					return
				}
				deps, err := cookbook.DependsCookbooks(cb_ver["run_list"].([]string))
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusMethodNotAllowed)
				}
				/* Need our own encoding here too. */
				enc := json.NewEncoder(w)
				if err := enc.Encode(&deps); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				}
				return
			case "cookbooks":
				env_response = env.AllCookbookHash(num_results)
			case "nodes":
				node_list := node.GetList()
				for _, n := range node_list {
					chef_node, _ := node.Get(n)
					if chef_node == nil {
						continue
					}
					if chef_node.ChefEnvironment == env_name {
						env_response[chef_node.Name] = util.ObjURL(chef_node) 
					}
				}
			case "recipes":
				env_recipes := env.RecipeList()
				/* And... we have to do our own json response
				 * here. Hmph. */
				/* TODO: make the JSON encoding stuff its own
				 * function. Dunno why I never thought of that
				 * before now for this. */
				enc := json.NewEncoder(w)
				if err := enc.Encode(&env_recipes); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				}
				return
			default:
				JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return

		}
	} else if path_array_len == 4 {
		env_name := path_array[1]
		/* op is either "cookbooks" or "roles", and op_name is the name
		 * of the object op refers to. */
		op := path_array[2]
		op_name := path_array[3]

		if r.Method != "GET" {
			JsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		/* Redirect op=roles to /roles/NAME/environments/NAME. The API
		 * docs recommend but do not require using that URL, so in the
		 * interest of simplicity we will just redirect to it. */
		if op == "roles" {
			redir_url := fmt.Sprintf("/roles/%s/environments/%s", op_name, env_name)
			http.Redirect(w, r, redir_url, http.StatusMovedPermanently)
			return
		} else if op == "cookbooks" {
			env, err := environment.Get(env_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			cb, err := cookbook.Get(op_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			env_response[op_name] = cb.ConstrainedInfoHash(num_results, env.CookbookVersions[op_name])
		} else {
			/* Not an op we know. */
			JsonErrorReport(w, r, "Bad request - too many elements in path", http.StatusBadRequest)
			return
		}
	} else {
		/* Bad number of path elements. */
		JsonErrorReport(w, r, "Bad request - too many elements in path", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&env_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
