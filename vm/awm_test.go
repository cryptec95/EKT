package vm

import (
	"github.com/EducationEKT/EKT/db"
	"testing"
)

func TestDB(t *testing.T) {
	vm := NewVM(nil, db.NewComposedKVDatabase("/home/x/test/ekt/db"))
	vm.Run(`
		console.log(!!AWM.db_get("hello"));
		console.log(AWM.db_set(123, "world"));
		console.log(AWM.db_get("123"));
		//console.log(AWM.db_delete("hello"));
		//console.log(!AWM.db_get("hello"));
	`)
}
