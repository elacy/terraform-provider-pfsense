package pfsense

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

const idSeparator = "."

type updateRequestFunc[RequestType any] func(*schema.ResourceData, string, *RequestType) error
type getFromResourceFunc[ResponseType any] func(*ResponseType) (interface{}, error)

type updateFunc[RequestType any, ResponseType any, IdType ~string | ~int] func(context.Context, *pfsenseapi.Client, IdType, *RequestType) (*ResponseType, error)
type createFunc[RequestType any, ResponseType any] func(context.Context, *pfsenseapi.Client, *RequestType) (*ResponseType, error)
type listFunc[ResponseType any] func(context.Context, *pfsenseapi.Client, string) ([]*ResponseType, error)
type deleteFunc[IdType ~string | ~int] func(context.Context, *pfsenseapi.Client, string, IdType) error
type disableFunc[RequestType any] func(*RequestType) error

var dnsValidator schema.SchemaValidateFunc = validation.StringMatch(regexValidator(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}$`), "Invalid DNS Name")
var hostNameValidator schema.SchemaValidateFunc = validation.StringMatch(regexValidator(`^[a-zA-Z0-9]+$`), "Invalid Host Name")

type resourceProperty[RequestType any, ResponseType any] struct {
	schema          *schema.Schema
	idProperty      bool
	partition       bool
	updateRequest   updateRequestFunc[RequestType]
	getFromResponse getFromResourceFunc[ResponseType]
	validValues     []string
}

type resource[RequestType any, ResponseType any, IdType ~string | ~int] struct {
	name        string
	description string
	getId       func(context.Context, *pfsenseapi.Client, *ResponseType) (IdType, error)
	partitionId string
	update      updateFunc[RequestType, ResponseType, IdType]
	create      createFunc[RequestType, ResponseType]
	delete      deleteFunc[IdType]
	disable     disableFunc[RequestType]
	list        listFunc[ResponseType]
	properties  map[string]*resourceProperty[RequestType, ResponseType]
}

func (r *resource[RequestType, ResponseType, IdType]) updateRequest(d *schema.ResourceData, request *RequestType) error {
	for name, prop := range r.properties {
		value, exists := d.GetOk(name)

		if exists {
			value = parseValue(value)

			if value == nil {
				exists = false
			}
		} else if prop.schema.Default != nil {
			d.Set(name, prop.schema.Default)
			value = prop.schema.Default
			exists = true
		}

		if exists {
			if err := prop.updateRequest(d, name, request); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *resource[RequestType, ResponseType, IdType]) updateResource(d *schema.ResourceData, response *ResponseType) error {
	for name, prop := range r.properties {
		if prop.getFromResponse == nil {
			continue
		}

		value, err := prop.getFromResponse(response)

		if err != nil {
			return err
		}

		value = parseValue(value)

		if value != nil {
			if err = d.Set(name, value); err != nil {
				return err
			}
		} else if prop.schema.Default != nil {
			d.Set(name, prop.schema.Default)
		}
	}

	return nil
}

func (r *resource[RequestType, ResponseType, IdType]) GetCreateFunction() schema.CreateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)
		request := new(RequestType)

		if err := r.updateRequest(d, request); err != nil {
			return diag.FromErr(err)
		}

		response, err := r.create(ctx, client, request)

		if err != nil {
			return diag.FromErr(err)
		}

		if err := r.updateResource(d, response); err != nil {
			return diag.FromErr(err)
		}

		id, err := r.getId(ctx, client, response)

		if err != nil {
			return diag.FromErr(err)
		}

		if reflect.ValueOf(id).IsZero() {
			return diag.Errorf("Invalid ID returned for %s: '%s'", r.name, fmt.Sprint(id))
		}

		if r.partitionId != "" {
			i, ok := d.GetOk(r.partitionId)

			if !ok {
				return diag.Errorf("Field %s is required, provider error, should be already validated", r.partitionId)
			}

			parition, ok := i.(string)

			if !ok {
				return diag.Errorf("Field %s should be a string, provider error, should be already validated", r.partitionId)
			}

			d.SetId(fmt.Sprintf("%s%s%s", parition, idSeparator, fmt.Sprint(id)))
		} else {
			d.SetId(fmt.Sprint(id))
		}

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) UpdateFromId(ctx context.Context, client *pfsenseapi.Client, d *schema.ResourceData) error {
	var list []*ResponseType
	var err error

	partition, id, err := r.getResourceId(d)

	if err != nil {
		return err
	}

	list, err = r.list(ctx, client, partition)

	if err != nil {
		return err
	}

	for _, item := range list {
		itemId, err := r.getId(ctx, client, item)

		if err != nil {
			return fmt.Errorf("Unable to get Id from listed value, received err: %v", err)
		}

		if id == itemId {
			if err = r.updateResource(d, item); err != nil {
				return err
			}

			if r.partitionId != "" {
				if err := d.Set(r.partitionId, partition); err != nil {
					return err
				}
			}

			return nil
		}
	}

	var partitionErrorText string

	if r.partitionId != "" {
		partitionErrorText = fmt.Sprintf(" and with %s equal to %s", r.partitionId, partition)
	}

	return fmt.Errorf("Unable to find item with Id %s%s", fmt.Sprint(id), partitionErrorText)

}

func (r *resource[RequestType, ResponseType, IdType]) GetReadFunction() schema.ReadContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)
		if err := r.UpdateFromId(ctx, client, d); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) getResourceId(d *schema.ResourceData) (string, IdType, error) {
	var partition string
	id := d.Id()

	if r.partitionId != "" {
		parts := splitIntoArray(id, idSeparator)
		partition = parts[0]
		id = strings.Join(parts[1:], idSeparator)
	}

	var example interface{} = new(IdType)
	var zeroValue IdType

	switch example.(type) {
	case *int:
		number, err := strconv.Atoi(id)

		if err != nil {
			return "", zeroValue, nil
		}

		value := any(number).(IdType)

		return partition, value, nil
	case *string:
		value := any(id).(IdType)
		return partition, value, nil
	}

	return "", zeroValue, fmt.Errorf("Unable to determine type of example %v", example)
}

func (r *resource[RequestType, ResponseType, IdType]) GetUpdateFunction() schema.UpdateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)
		request := new(RequestType)

		if err := r.updateRequest(d, request); err != nil {
			return diag.FromErr(err)
		}

		_, id, err := r.getResourceId(d)

		if err != nil {
			return diag.FromErr(err)
		}

		response, err := r.update(ctx, client, id, request)

		if err != nil {
			return diag.FromErr(err)
		}

		if err = r.updateResource(d, response); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) GetDeleteFunction() schema.DeleteContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)

		partition, id, err := r.getResourceId(d)

		if err != nil {
			return diag.FromErr(err)
		}

		if r.delete != nil {
			err := r.delete(ctx, client, partition, id)

			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err := r.UpdateFromId(ctx, client, d); err != nil {
				return diag.FromErr(err)
			}

			request := new(RequestType)

			if err := r.updateRequest(d, request); err != nil {
				return diag.FromErr(err)
			}

			if err := r.disable(request); err != nil {
				return diag.FromErr(err)
			}

			if _, err := r.update(ctx, client, id, request); err != nil {
				return diag.FromErr(err)
			}
		}

		d.SetId("")

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) GetDiffSupressFunction(property *resourceProperty[RequestType, ResponseType]) schema.SchemaDiffSuppressFunc {
	if property.schema.Default == nil || property.schema.Default == "" {
		return nil
	}

	return func(k, oldValue, newValue string, d *schema.ResourceData) bool {
		return (oldValue == property.schema.Default || newValue == property.schema.Default) && (oldValue == "" || newValue == "")
	}
}

func (r *resource[RequestType, ResponseType, IdType]) GetImporter() *schema.ResourceImporter {
	return &schema.ResourceImporter{
		StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
			client := m.(*pfsenseapi.Client)

			if err := r.UpdateFromId(ctx, client, d); err != nil {
				return nil, err
			}

			return []*schema.ResourceData{d}, nil
		},
	}
}

func (r *resource[RequestType, ResponseType, IdType]) AddResource(provider *schema.Provider) {
	_, exists := provider.ResourcesMap[r.name]

	if exists {
		panic(fmt.Sprintf("Resource %s already exists", r.name))
	}

	resource := &schema.Resource{
		CreateContext: r.GetCreateFunction(),
		ReadContext:   r.GetReadFunction(),
		UpdateContext: r.GetUpdateFunction(),
		DeleteContext: r.GetDeleteFunction(),
		Importer:      r.GetImporter(),
		Schema:        map[string]*schema.Schema{},
		Description:   r.description,
	}

	var idName string

	for name, property := range r.properties {

		if property.idProperty {
			idName = name
		} else if property.partition {
			r.partitionId = name
		}

		resource.Schema[name] = property.schema
		resource.Schema[name].DiffSuppressFunc = r.GetDiffSupressFunction(property)
	}

	if idName != "" {
		if r.getId != nil {
			panic(fmt.Sprintf("Shouldn't have get ID function set and an id property, provider error on %s", r.name))
		}

		r.getId = func(_ context.Context, _ *pfsenseapi.Client, response *ResponseType) (IdType, error) {
			var zeroValue IdType
			i, err := r.properties[idName].getFromResponse(response)

			if err != nil {
				return zeroValue, err
			}

			if i == zeroValue {
				return zeroValue, fmt.Errorf("Retrieved '%s' as an property %s", i, idName)
			}

			id, ok := i.(IdType)

			if !ok {
				return zeroValue, fmt.Errorf("Unable to convert %v (value of %s) to IdType", i, idName)
			}

			return id, nil
		}
	}

	provider.ResourcesMap[r.name] = resource
}
