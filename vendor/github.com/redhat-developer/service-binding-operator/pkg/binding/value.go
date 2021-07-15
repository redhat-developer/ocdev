package binding

type Value interface {
	Get() interface{}
}

type value struct {
	v interface{}
}

var _ Value = (*value)(nil)

func (v *value) Get() interface{} {
	return v.v
}
