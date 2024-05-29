package strategy

type PersistenceStrategy interface {
	Set(key string, value interface{}) error
	Get(key string, dest interface{}) error
	GetCount(key string) int
	Delete(key string) error
}
