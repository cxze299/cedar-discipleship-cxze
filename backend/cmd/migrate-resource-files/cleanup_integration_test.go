//go:build integration

package main

import (
	"testing"

	"agp/backend/internal/asset"
	"agp/backend/internal/testdb"
)

func TestCleanupDuplicateResourceBindings(t *testing.T) {
	for _, hasConsumer := range []bool{false, true} {
		name := "unreferenced duplicates"
		if hasConsumer {
			name = "imported source identity"
		}
		t.Run(name, func(t *testing.T) {
			db := testdb.Open(t)
			testdb.Exec(t, db, `INSERT INTO study_groups(id,code,name,created_at,updated_at)
				VALUES (1,'a','A',NOW(),NOW()),(2,'b','B',NOW(),NOW());
				INSERT INTO assets(id,group_id,category,title,original_name,storage_path,file_size,checksum_sha256,visibility,created_by,created_at,updated_at)
				VALUES (1,1,'book','Lesson 1-2页','Lesson.pdf','team-a-resources/objects/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/Lesson.pdf',20,REPEAT('a',64),'all_groups',1,NOW(),NOW()),
				       (2,1,'book','Lesson','Lesson.pdf','team-a-resources/objects/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/Lesson.pdf',20,REPEAT('a',64),'group',1,NOW(),NOW());
				INSERT INTO asset_bindings(asset_id,group_id,resource_key,asset_kind,created_at,updated_at)
				VALUES (1,1,REPEAT('a',32),'owned',NOW(),NOW()),(2,1,REPEAT('b',32),'owned',NOW(),NOW());
				INSERT INTO asset_share_grants(asset_id,owner_group_id,consumer_group_id,status,created_by,created_at)
				VALUES (1,1,NULL,'active',1,NOW())`)
			if hasConsumer {
				testdb.Exec(t, db, `INSERT INTO assets(id,group_id,category,title,original_name,storage_path,visibility,created_by,created_at,updated_at)
					VALUES (3,2,'book','Imported','Lesson.pdf','team-a-resources/objects/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/Lesson.pdf','imported',1,NOW(),NOW());
					INSERT INTO asset_bindings(asset_id,group_id,resource_key,asset_kind,source_asset_id,created_at,updated_at)
					VALUES (3,2,REPEAT('c',32),'imported',1,NOW(),NOW());
					INSERT INTO asset_dependencies(consumer_group_id,consumer_asset_id,provider_group_id,provider_asset_id,status,created_at,updated_at)
					VALUES (2,3,1,1,'active',NOW(),NOW())`)
				if _, err := asset.NewMySQLRepository(db).FindDownloadTarget(t.Context(), 2, 3); err != nil {
					t.Fatalf("fixture download: %v", err)
				}
			}
			want := 1
			if hasConsumer {
				want = 0
			}
			for _, dryRun := range []bool{true, false} {
				n, err := cleanupDuplicateResourceBindings(t.Context(), db, 1, dryRun)
				if err != nil {
					t.Fatal(err)
				}
				if n != want {
					t.Errorf("dryRun=%v removed=%d want=%d", dryRun, n, want)
				}
			}
			if hasConsumer {
				if _, err := asset.NewMySQLRepository(db).FindDownloadTarget(t.Context(), 2, 3); err != nil {
					t.Fatalf("cleanup broke existing import: %v", err)
				}
				var dependencies, grants int
				if err := db.QueryRow(`SELECT COUNT(*) FROM asset_dependencies WHERE provider_asset_id=1 AND status='active'`).Scan(&dependencies); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow(`SELECT COUNT(*) FROM asset_share_grants WHERE asset_id=1 AND status='active'`).Scan(&grants); err != nil {
					t.Fatal(err)
				}
				if dependencies != 1 || grants != 1 {
					t.Fatalf("authority changed: dependencies=%d grants=%d", dependencies, grants)
				}
			}
		})
	}
}
