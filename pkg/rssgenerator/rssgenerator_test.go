package rssgenerator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestResultGenerateHash(t *testing.T) {
	res := Result{
		Secret:  &corev1.Secret{},
		Service: &corev1.Service{},
		STS:     &appsv1.StatefulSet{},
	}

	assert.NoError(t, res.generateHash(), "should be able to generate hash")

	compHash := res.Hash
	assert.NoError(t, res.generateHash(), "should be able to generate hash (again)")
	assert.Equal(t, compHash, res.Hash, "same content should have same hash")

	res = Result{
		Secret:  &corev1.Secret{},
		Service: &corev1.Service{},
		STS:     &appsv1.StatefulSet{},
	}

	assert.NoError(t, res.generateHash(), "should be able to generate hash")
	assert.Equal(t, compHash, res.Hash, "equal content should have same hash (ignoring pointer addresses)")

	res.Secret.Namespace = "foobar"
	assert.NoError(t, res.generateHash(), "should be able to generate hash (again)")
	assert.NotEqual(t, compHash, res.Hash, "modified content should NOT have same hash")
}
