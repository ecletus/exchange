package exchange

import (
	"fmt"
	"reflect"

	"github.com/ecletus/plug"

	"github.com/moisespsena/go-edis"
	"github.com/moisespsena-go/path-helpers"

	"github.com/ecletus/core"
	"github.com/ecletus/core/resource"
	"github.com/ecletus/roles"
	"github.com/ecletus/validations"
	"github.com/moisespsena-go/aorm"
)

var PKG = path_helpers.GetCalledDir()

// Resource defined an exchange resource, which includes importing/exporting fields definitions
type Resource struct {
	*resource.Resource
	Config *Config
	Metas  []*Meta
}

type Exchange struct {
	edis.EventDispatcher

	FakeDB    *aorm.DB
	resources map[string]*Resource
}

// Config is exchange resource config
type Config struct {
	// PrimaryField that used as primary field when searching resource from database
	PrimaryField string
	// Permission defined permission
	Permission *roles.Permission
	// WithoutHeader no header in the data file
	WithoutHeader      bool
	DisableTransaction bool
}

// NewResource new exchange Resource
func (e *Exchange) NewResource(value interface{}, config ...Config) *Resource {
	res := Resource{Resource: resource.New(e.FakeDB.NewScope(value), "", "")}

	if len(config) > 0 {
		res.Config = &config[0]
	} else {
		res.Config = &Config{}
	}
	res.Permission = res.Config.Permission

	if res.Config.PrimaryField != "" {
		if err := res.SetPrimaryFields(res.Config.PrimaryField); err != nil {
			fmt.Println(err)
		}
	}
	return &res
}

func (e *Exchange) AddResource(key string, value interface{}, config ...Config) (res *Resource) {
	res = e.NewResource(value, config...)
	e.resources[key] = res
	e.Trigger(&ResourceAddedEvent{plug.NewEvent(E_RESOURCE_ADDED), e, res})
	return
}

// Meta define exporting/importing meta for exchange Resource
func (res *Resource) Meta(meta *Meta) *Meta {
	if meta.Header == "" {
		meta.Header = meta.Name
	}

	meta.base = res
	meta.updateMeta()
	res.Metas = append(res.Metas, meta)
	return meta
}

// GetMeta get defined Meta from exchange Resource
func (res *Resource) GetMeta(name string) *Meta {
	for _, meta := range res.Metas {
		if meta.Header == name {
			return meta
		}
	}
	return nil
}

// GetMetas get all defined Metas from exchange Resource
func (res *Resource) GetMetas([]string) []resource.Metaor {
	metas := []resource.Metaor{}
	for _, meta := range res.Metas {
		metas = append(metas, meta)
	}
	return metas
}

type errorsInterface interface {
	GetErrors() []error
}

// Import used to import data into a exchange Resource
//     product.Import(csv.New("products.csv"), context)
func (res *Resource) Import(container Container, context *core.Context, callbacks ...func(Progress) error) error {
	rows, err := container.NewReader(res, context)
	if err == nil {
		var hasError bool
		var current uint
		var total = rows.Total()

		if db := context.GetDB(); db != nil && !res.Config.DisableTransaction {
			tx := db.Begin()
			context.SetDB(tx)
			defer func() {
				if hasError {
					tx.Rollback()
				} else {
					tx.Commit()
				}
			}()
		}

		for rows.Next() {
			current++
			progress := Progress{Total: total, Current: current}

			var metaValues *resource.MetaValues
			var handleError func(err error)

			if metaValues, err = rows.ReadRow(); err == nil {
				for _, metaValue := range metaValues.Values {
					progress.Cells = append(progress.Cells, Cell{
						Header: metaValue.Name,
						Value:  metaValue.Value,
					})
				}

				handleError = func(err error) {
					hasError = true
					progress.Errors.AddError(err)

					if errors, ok := err.(errorsInterface); ok {
						for _, err := range errors.GetErrors() {
							handleError(err)
						}
					} else if err, ok := err.(*validations.Error); ok {
						for idx, cell := range progress.Cells {
							if cell.Header == err.Column {
								cell.Error = err
								progress.Cells[idx] = cell
								break
							}
						}
					} else if len(progress.Cells) > 0 {
						var err error = err
						cell := progress.Cells[0]
						if cell.Error != nil {
							var errors core.Errors
							errors.AddError(cell.Error)
							errors.AddError(err)
							err = errors
						}
						cell.Error = err
					}
				}

				result := res.NewStruct(context.Site)
				progress.Value = result
				crud := res.Crud(context)

				if err = crud.SetMetaValues(metaValues).FindOne(result); err == nil || aorm.IsRecordNotFoundError(err) {
					if err = resource.DecodeToResource(res, result, metaValues, context).Start(); err == nil {
						if err = crud.SaveOrCreate(result); err != nil {
							handleError(err)
						}
					} else {
						handleError(err)
					}
				} else {
					handleError(err)
				}
			}

			for _, callback := range callbacks {
				if err := callback(progress); err != nil {
					return err
				}
			}
		}
	}
	return err
}

// Export used export data from a exchange Resource
//     product.Export(csv.New("products.csv"), context)
func (res *Resource) Export(container Container, context *core.Context, callbacks ...func(Progress) error) error {
	var (
		total   uint
		results = res.NewSlice()
		err     = context.GetDB().Find(results).Count(&total).Error
	)

	if err == nil {
		reflectValue := reflect.Indirect(reflect.ValueOf(results))

		writer, err := container.NewWriter(res, context)

		if err == nil {
			writer.WriteHeader()

			for i := 0; i < reflectValue.Len(); i++ {
				var result = reflectValue.Index(i).Interface()
				var metaValues *resource.MetaValues
				if metaValues, err = writer.WriteRow(result); err != nil {
					return err
				}

				var progress = Progress{
					Current: uint(i + 1),
					Total:   total,
					Value:   result,
				}

				for _, metaValue := range metaValues.Values {
					progress.Cells = append(progress.Cells, Cell{
						Header: metaValue.Name,
						Value:  metaValue.Value,
					})
				}

				for _, callback := range callbacks {
					if err := callback(progress); err != nil {
						return err
					}
				}
			}
			err = writer.Flush()
		}

		return err
	}

	return err
}
