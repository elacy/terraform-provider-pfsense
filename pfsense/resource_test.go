package pfsense

import (
	"context"
	"fmt"
	"testing"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

type resourceTest interface {
	RunTests(t *testing.T)
	GetName() string
}

type convertFunc[RequestType any, ResponseType any, IdType ~string | ~int] func(*RequestType) (*ResponseType, error)
type resourceTestFunc[RequestType any, ResponseType any, IdType ~string | ~int] func(t *testing.T)

type tfResourceTest[RequestType any, ResponseType any, IdType ~string | ~int] struct {
	resource     *resource[RequestType, ResponseType, IdType]
	convert      convertFunc[RequestType, ResponseType, IdType]
	currentState map[string][]*ResponseType
	getPartition func(*RequestType) string
	fuzzer       fuzz.ConsumeFuzzer
	provider     *schema.Provider
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) GetName() string {
	return r.resource.name
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) RunTests(t *testing.T) {
	r.currentState = map[string][]*ResponseType{}
	r.provider = Provider()

	delete(r.provider.ResourcesMap, r.resource.name)
	r.resource.AddResource(r.provider)

	testFuncs := map[string]resourceTestFunc[RequestType, ResponseType, IdType]{
		"noMoreThanOneId":        r.noMoreThanOneId,
		"noMoreThanOnePartition": r.noMoreThanOnePartition,
		"idIsRequired":           r.idIsRequired,
		"partitionIsRequired":    r.partitionIsRequired,
		"partitionIsForceNew":    r.partitionIsForceNew,
		"getIdIsSet":             r.getIdIsSet,
		"functionsAreSet":        r.functionsAreSet,
	}

	for name, testFunc := range testFuncs {
		t.Run(fmt.Sprintf("%s::%s", r.resource.name, name), func(t *testing.T) {
			testFunc(t)
		})
	}

	r.resource.create = r.create
	r.resource.list = r.list
	r.resource.update = r.update

	if r.resource.delete != nil {
		r.resource.delete = r.delete
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) noMoreThanOneId(t *testing.T) {
	i := 0

	for _, property := range r.resource.properties {
		if property.idProperty {
			i++
		}
	}

	if i > 1 {
		t.Errorf("Should be no more than one ID properties but found %d on %s", i, r.resource.name)
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) noMoreThanOnePartition(t *testing.T) {
	i := 0

	for _, property := range r.resource.properties {
		if property.partition {
			i++
		}
	}

	if i > 1 {
		t.Errorf("Not more than one partition allowed, found %d on %s", i, r.resource.name)
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) idIsRequired(t *testing.T) {
	for name, property := range r.resource.properties {
		if property.idProperty && !property.schema.Required {
			t.Errorf("Property %s on resource %s is an ID but it's not required", name, r.resource.name)
		}
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) partitionIsRequired(t *testing.T) {
	for name, property := range r.resource.properties {
		if property.partition && !property.schema.Required {
			t.Errorf("Property %s on resource %s is a partition but it's not required", name, r.resource.name)
		}
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) partitionIsForceNew(t *testing.T) {
	for name, property := range r.resource.properties {
		if property.partition && !property.schema.ForceNew {
			t.Errorf("Property %s on resource %s is a partition but it's not force new", name, r.resource.name)
		}
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) getIdIsSet(t *testing.T) {
	if r.resource.getId == nil {
		t.Errorf("Get ID function is not set on resource %s", r.resource.name)
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) functionsAreSet(t *testing.T) {
	if r.resource.create == nil {
		t.Errorf("Create Function is not set on resource %s", r.resource.name)
	}

	if r.resource.delete == nil && r.resource.disable == nil {
		t.Errorf("Delete/Disable Function is not set on resource %s", r.resource.name)
	} else if r.resource.delete != nil && r.resource.disable != nil {
		t.Errorf("Both Delete and Disable Function is set on resource %s", r.resource.name)
	}

	if r.resource.list == nil {
		t.Errorf("List Function is not set on resource %s", r.resource.name)
	}

	if r.resource.update == nil {
		t.Errorf("Update Function is not set on resource %s", r.resource.name)
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) create(_ context.Context, _ *pfsenseapi.Client, request *RequestType) (*ResponseType, error) {
	result, err := r.convert(request)

	if err != nil {
		return nil, err
	}

	var partition string
	if r.getPartition != nil {
		partition = r.getPartition(request)
	}

	r.initPartition(partition)
	r.currentState[partition] = append(r.currentState[partition], result)

	return result, nil
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) initPartition(partition string) {
	_, ok := r.currentState[partition]

	if !ok {
		r.currentState[partition] = []*ResponseType{}
	}
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) update(_ context.Context, _ *pfsenseapi.Client, id IdType, request *RequestType) (*ResponseType, error) {
	result, err := r.convert(request)

	if err != nil {
		return nil, err
	}

	var partition string
	if r.getPartition != nil {
		partition = r.getPartition(request)
	}

	if err != nil {
		return nil, err
	}

	r.initPartition(partition)

	for i, item := range r.currentState[partition] {
		itemId, err := r.resource.getId(item)

		if err != nil {
			return nil, err
		}

		if id == itemId {
			r.currentState[partition][i] = result
			return result, nil
		}
	}

	return nil, fmt.Errorf("Test error, unable to find Id %v within partition %s on resource %s", id, partition, r.resource.name)
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) delete(_ context.Context, _ *pfsenseapi.Client, partition string, id IdType) error {
	r.initPartition(partition)

	for i, item := range r.currentState[partition] {
		itemId, err := r.resource.getId(item)

		if err != nil {
			return err
		}

		if id == itemId {
			r.currentState[partition] = append(r.currentState[partition][:i], r.currentState[partition][i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("Test error, unable to find Id %v within partition %s on resource %s", id, partition, r.resource.name)
}

func (r *tfResourceTest[RequestType, ResponseType, IdType]) list(_ context.Context, _ *pfsenseapi.Client, partition string) ([]*ResponseType, error) {
	r.initPartition(partition)
	return r.currentState[partition], nil
}
