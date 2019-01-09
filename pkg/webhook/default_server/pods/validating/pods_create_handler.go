/*
Copyright 2018 Kaizhe Huang.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validating

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"image-scan-k8s-webhook/pkg/webhook/default_server/pods/anchore"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

var (
	log = logrus.New()
)

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})

	webhookName := "validating-create-pods"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &PodCreateHandler{})
}

// PodCreateHandler handles Pod
type PodCreateHandler struct {
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *PodCreateHandler) validatingPodFn(ctx context.Context, obj *corev1.Pod) (bool, string, error) {
	// TODO(user): implement your admission logic
	allowed := true
	msg := "allowed to be admitted"

	for _, container := range obj.Spec.Containers {
		image := container.Image
		log.Info("Checking image: " + image)
		if !anchore.CheckImage(image) {
			allowed = false
			msg = fmt.Sprintf("Image failed policy check: %s", image)
			log.Warning(msg)
			return allowed, msg, nil
		} else {
			log.Info("Image passed policy check: " + image)
		}
	}

	return allowed, msg, nil
}

var _ admission.Handler = &PodCreateHandler{}

// Handle handles admission requests.
func (h *PodCreateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &corev1.Pod{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingPodFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.Client = &PodCreateHandler{}
//
//// InjectClient injects the client into the PodCreateHandler
//func (h *PodCreateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &PodCreateHandler{}

// InjectDecoder injects the decoder into the PodCreateHandler
func (h *PodCreateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
