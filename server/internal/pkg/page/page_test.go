package page

import "testing"

// 边界套件：本包是钳制唯一 owner，各 List 不再自备 case。
// 最怕的失败：缺省/上限数字漂移（改一处漏一处），以及 0/-1/超大值穿透到 SQL。
func TestClampPage(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 1},
		{-1, 1},
		{-1000000, 1},
		{1, 1},
		{2, 2},
		{1000000000, 1000000000},
	}
	for _, c := range cases {
		if got := ClampPage(c.in); got != c.want {
			t.Errorf("ClampPage(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestClampPageSize(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 10},
		{-1, 10},
		{1, 1},
		{10, 10},
		{50, 50},
		{51, 10},
		{100, 10},
		{1000000000, 10},
	}
	for _, c := range cases {
		if got := ClampPageSize(c.in); got != c.want {
			t.Errorf("ClampPageSize(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestAuditClampSize(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 20},
		{-1, 20},
		{20, 20},
		{50, 50},
		{100, 100},
		{101, 20},
		{1000000000, 20},
	}
	for _, c := range cases {
		if got := Audit.ClampSize(c.in); got != c.want {
			t.Errorf("Audit.ClampSize(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestClamp(t *testing.T) {
	page, size := Clamp(0, 99)
	if page != 1 || size != 10 {
		t.Errorf("Clamp(0, 99) = (%d, %d), want (1, 10)", page, size)
	}
	page, size = Clamp(3, 25)
	if page != 3 || size != 25 {
		t.Errorf("Clamp(3, 25) = (%d, %d), want (3, 25)", page, size)
	}
}

func TestLimits(t *testing.T) {
	if Standard.Default != 10 || Standard.Max != 50 {
		t.Errorf("Standard = %+v, want {10 50}（契约 §1.3）", Standard)
	}
	if Audit.Default != 20 || Audit.Max != 100 {
		t.Errorf("Audit = %+v, want {20 100}（契约 §16 审计/重置记录）", Audit)
	}
}
