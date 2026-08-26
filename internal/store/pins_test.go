package store

import "testing"

func TestPinsCRUD(t *testing.T) {
	s := openTest(t)
	p, err := s.CreatePin("  Hello  ", []string{"#Foo", "foo", "Bar"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Hello" {
		t.Fatalf("title = %q", p.Title)
	}
	if len(p.Tags) != 2 || p.Tags[0] != "foo" || p.Tags[1] != "bar" {
		t.Fatalf("tags = %v", p.Tags)
	}

	if _, err := s.CreatePin("  ", nil, ""); err == nil {
		t.Fatal("empty title accepted")
	}

	got, err := s.GetPin(p.ID)
	if err != nil || got.Body != "body" {
		t.Fatalf("get = %+v %v", got, err)
	}

	upd, err := s.UpdatePin(p.ID, "Bye", []string{"x"}, "next")
	if err != nil || upd.Title != "Bye" || upd.Body != "next" {
		t.Fatalf("update = %+v %v", upd, err)
	}

	list, err := s.ListPins()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v %v", list, err)
	}

	if err := s.DeletePin(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetPin(p.ID); err != ErrNotFound {
		t.Fatalf("deleted get = %v", err)
	}
}

func TestPinFiles(t *testing.T) {
	s := openTest(t)
	p, err := s.CreatePin("Shot", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPinFile(p.ID, "x.exe", "application/octet-stream", 10); err == nil {
		t.Fatal("exe allowed")
	}
	f, err := s.AddPinFile(p.ID, "shot.png", "image/png", 120)
	if err != nil || f.Kind != "image" {
		t.Fatalf("image = %+v %v", f, err)
	}
	got, err := s.GetPin(p.ID)
	if err != nil || got.FileCount != 1 || len(got.Files) != 1 {
		t.Fatalf("get files = %+v %v", got, err)
	}
	list, _ := s.ListPins()
	if list[0].FileCount != 1 {
		t.Fatalf("list count = %d", list[0].FileCount)
	}
	if err := s.DeletePinFile(p.ID, f.ID); err != nil {
		t.Fatal(err)
	}
	sk, err := s.AddPinSketch(p.ID, "Board", "blank", "", 200)
	if err != nil || sk.Kind != "sketch" {
		t.Fatalf("sketch = %+v %v", sk, err)
	}
}
