//go:build integration

package backup

import (
	"testing"

	"agp/backend/internal/testdb"
)

func TestBackupAssetCandidates(t *testing.T) {
	db := testdb.Open(t)
	testdb.Exec(t, db, `INSERT INTO assets(id,group_id,category,title,original_name,storage_path,created_by,created_at,updated_at)
		VALUES (1,1,'book','Shared Book','shared.pdf','team-a-resources/objects/a/shared.pdf',1,NOW(),NOW()),
		       (2,1,'book','Ambiguous Book','first.pdf','team-a-resources/objects/b/first.pdf',1,NOW(),NOW()),
		       (3,1,'book','Ambiguous Book','second.pdf','team-a-resources/objects/c/second.pdf',1,NOW(),NOW()),
		       (4,1,'book','Shared Book','deleted.pdf','team-a-resources/objects/d/deleted.pdf',1,NOW(),NOW()),
		       (5,2,'book','Shared Book','foreign.pdf','team-b-resources/objects/e/foreign.pdf',1,NOW(),NOW()),
		       (6,1,'video','Shared Book','video.mp4','team-a-resources/objects/f/video.mp4',1,NOW(),NOW());
		INSERT INTO asset_bindings(asset_id,group_id,resource_key,asset_kind,deleted_at,created_at,updated_at)
		VALUES (1,1,'a','owned',NULL,NOW(),NOW()),(2,1,'b','owned',NULL,NOW(),NOW()),
		       (3,1,'c','owned',NULL,NOW(),NOW()),(4,1,'d','owned',NOW(),NOW(),NOW()),
		       (5,2,'e','owned',NULL,NOW(),NOW()),(6,1,'f','owned',NULL,NOW(),NOW())`)
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	selectCount := func() int {
		t.Helper()
		var name string
		var count int
		if err := tx.QueryRowContext(t.Context(), "SHOW SESSION STATUS LIKE 'Com_select'").Scan(&name, &count); err != nil {
			t.Fatal(err)
		}
		return count
	}
	cache := make(backupTaskAssetCandidates)
	before := selectCount()
	for _, tc := range []struct {
		category string
		refs     []string
		want     uint64
	}{
		{category: "book", refs: []string{"Shared Book 1-5页", "Unmatched"}, want: 1},
		{category: "book", refs: []string{"Shared Book 5-10页"}, want: 1},
		{category: "book", refs: []string{"Ambiguous Book"}, want: 0},
		{category: "video", refs: []string{"Shared Book"}, want: 6},
		{category: "outline", refs: []string{"Missing"}, want: 0},
		{category: "outline", refs: []string{"Still Missing"}, want: 0},
	} {
		id, err := findBackupTaskAssetByReferenceTx(t.Context(), tx, 1, tc.category, tc.refs, cache)
		if err != nil || id != tc.want {
			t.Fatalf("%s %v: id=%d want=%d err=%v", tc.category, tc.refs, id, tc.want, err)
		}
	}
	if queries := selectCount() - before; queries != 3 {
		t.Fatalf("candidate selects=%d want one per category (3)", queries)
	}
}
