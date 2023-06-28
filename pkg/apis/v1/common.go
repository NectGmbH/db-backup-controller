package v1

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type (
	// Secret contains an optional Value or reference to fetch the
	// value FromSecret
	Secret struct {
		// Value specifies a plain text value for the secret. When filled
		// this will prevent the lookup of the FromSecret reference.
		//
		// +kubebuilder:validation:Optional
		Value string `json:"value"`
		// FromSecret references a secret to fetch the value from
		//
		// +kubebuilder:validation:Optional
		FromSecret SecretKeyRef `json:"fromSecret"`
	}

	// SecretKeyRef contains information where to find the Secret
	// information when fetching from cluster
	SecretKeyRef struct {
		// Name specifies the name of the secret to fetch the value from.
		// Must exist in the same namespace as the resource
		Name string `json:"name"`
		// Key specifies the key within the refereced secret to fetch the
		// value from
		Key string `json:"key"`
	}
)

// CopyFromSecret fetches the secret given in the reference and
// copies the content of the key into the Value
func (s *Secret) CopyFromSecret(ctx context.Context, client kubernetes.Interface, namespace string) error {
	if s.Value != "" {
		// Value is set, no need to copy
		return nil
	}

	if s.FromSecret.Name == "" || s.FromSecret.Key == "" {
		// Nothing to copy
		return nil
	}

	secret, err := client.CoreV1().
		Secrets(namespace).
		Get(ctx, s.FromSecret.Name, metaV1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "fetching secret")
	}

	data, ok := secret.Data[s.FromSecret.Key]
	if !ok {
		return errors.Errorf("key %q not found in secret", s.FromSecret.Key)
	}

	s.Value = string(data) // NOTE(kahlers): Something magically b64-decodes this, so this is fine

	return nil
}

//nolint:gocognit,gocyclo // Yipp, that's complex. Thats reflect magic. Not gonna split.
func fetchSecretsRecurse(ctx context.Context, in any, client kubernetes.Interface, namespace string) (err error) {
	var (
		secretType = reflect.TypeOf(Secret{})
		vo, to     = reflect.ValueOf(in), reflect.TypeOf(in)
	)

	if to.Kind() != reflect.Ptr {
		return errors.New("called on non-pointer")
	}

	if vo.IsNil() {
		// We don't need to check nil values
		return nil
	}

	if vo.Elem().Kind() != reflect.Struct {
		return errors.New("called on pointer to non-struct")
	}

	st := vo.Elem()
	for i := 0; i < st.NumField(); i++ {
		valField := st.Field(i)
		typeField := st.Type().Field(i)

		switch {
		case typeField.Type == secretType:
			if err = valField.Addr().Interface().(*Secret).CopyFromSecret(ctx, client, namespace); err != nil {
				return errors.Wrapf(err, "fetching secrets for %s", typeField.Name)
			}

		case typeField.Type.Kind() == reflect.Ptr && !valField.IsNil() && valField.Elem().Type() == secretType:
			if err = valField.Elem().Addr().Interface().(*Secret).CopyFromSecret(ctx, client, namespace); err != nil {
				return errors.Wrapf(err, "fetching secrets for %s", typeField.Name)
			}

		case typeField.Type.Kind() == reflect.Struct:
			if err = fetchSecretsRecurse(ctx, valField.Addr().Interface(), client, namespace); err != nil {
				return errors.Wrapf(err, "fetching secrets in %s", typeField.Name)
			}

		case typeField.Type.Kind() == reflect.Ptr && valField.Elem().Kind() == reflect.Struct:
			if err = fetchSecretsRecurse(ctx, valField.Elem().Addr().Interface(), client, namespace); err != nil {
				return errors.Wrapf(err, "fetching secrets in %s", typeField.Name)
			}

		case typeField.Type.Kind() == reflect.Slice && typeField.Type.Elem().Kind() == reflect.Struct:
			for i := 0; i < valField.Len(); i++ {
				if err = fetchSecretsRecurse(ctx, valField.Index(i).Addr().Interface(), client, namespace); err != nil {
					return errors.Wrapf(err, "fetching secrets in %s (idx %d)", typeField.Name, i)
				}
			}

		default:
			// We don't care about that field
			continue
		}
	}

	return nil
}
