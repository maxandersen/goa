package middleware_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ErrorHandler", func() {
	var service *goa.Service
	var h goa.Handler
	var suppressInternal bool

	var rw *testResponseWriter

	BeforeEach(func() {
		service = nil
		h = nil
		suppressInternal = false
		rw = nil
	})

	JustBeforeEach(func() {
		rw = newTestResponseWriter()
		eh := middleware.ErrorHandler(suppressInternal)(h)
		req, err := http.NewRequest("GET", "/foo", nil)
		Ω(err).ShouldNot(HaveOccurred())
		ctx := newContext(service, rw, req, nil)
		err = eh(ctx, rw, req)
		Ω(err).ShouldNot(HaveOccurred())
	})

	Context("with a handler returning a Go error", func() {
		BeforeEach(func() {
			service = newService(nil)
			h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
				return errors.New("boom")
			}
		})

		It("turns Go errors into HTTP 500 responses", func() {
			Ω(rw.Status).Should(Equal(500))
			Ω(rw.ParentHeader["Content-Type"]).Should(Equal([]string{"text/plain"}))
			Ω(string(rw.Body)).Should(Equal(`"boom"` + "\n"))
		})

		Context("suppressing internal errors", func() {
			BeforeEach(func() {
				suppressInternal = true
			})

			It("suppresses the error details", func() {
				var decoded goa.Error
				Ω(rw.Status).Should(Equal(500))
				Ω(rw.ParentHeader["Content-Type"]).Should(Equal([]string{goa.ErrorMediaIdentifier}))
				err := service.Decode(&decoded, bytes.NewBuffer(rw.Body), "application/json")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(fmt.Sprintf("%v", decoded)).Should(Equal(fmt.Sprintf("%v", *goa.ErrInternal("internal error, detail suppressed"))))
			})

		})
	})

	Context("with a handler returning a goa error", func() {
		var gerr *goa.Error

		BeforeEach(func() {
			service = newService(nil)
			gerr = &goa.Error{
				Status:     418,
				Detail:     "teapot",
				Code:       "code",
				MetaValues: map[string]interface{}{"foobar": 42},
			}
			h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
				return gerr
			}
		})

		It("maps goa errors to HTTP responses", func() {
			var decoded goa.Error
			Ω(rw.Status).Should(Equal(gerr.Status))
			Ω(rw.ParentHeader["Content-Type"]).Should(Equal([]string{goa.ErrorMediaIdentifier}))
			err := service.Decode(&decoded, bytes.NewBuffer(rw.Body), "application/json")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(fmt.Sprintf("%v", decoded)).Should(Equal(fmt.Sprintf("%v", *gerr)))
		})
	})
})
