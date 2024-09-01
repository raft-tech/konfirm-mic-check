/*
 * Copyright (c) 2024 Raft, LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package mic

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handlers", func() {

	var server = NewHandler()

	It("GET /check", func() {

		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest("GET", "/check", nil))
		res := rec.Result()

		Expect(res).To(HaveHTTPStatus(http.StatusOK))
		Expect(res).To(HaveHTTPBody([]byte("Mic mic. One two. One two.")))
	})

	It("POST /replay", func() {

		// Create a 5 MiB body
		seed := []byte("All work and no play makes Jack a dull boy. 1337 1337 1337 1337\n")
		body := make([]byte, 5242880)
		for i := 0; i < 81920; i++ {
			copy(body[i*64:], seed)
		}

		// Send it
		req := httptest.NewRequest(http.MethodPost, "/replay", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "text/redrum;charset=haxor")
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		res := rec.Result()

		// Check it
		Expect(res).To(HaveHTTPStatus(http.StatusOK))
		Expect(res).To(HaveHTTPHeaderWithValue("Content-Type", "text/redrum;charset=haxor"))
		Expect(res).To(HaveHTTPHeaderWithValue("Content-Length", "5242880"))
		var err error
		body, err = io.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(body[5242816:]).To(Equal(seed)) // The last 64 bytes should equal the seed value
	})
})
