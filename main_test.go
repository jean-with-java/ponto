package main

import (
	"testing"
	"time"
)

func h(s string) time.Duration {
	d, err := parseHora(s)
	if err != nil {
		panic(err)
	}
	return d
}

func hs(ss ...string) []time.Duration {
	var out []time.Duration
	for _, s := range ss {
		out = append(out, h(s))
	}
	return out
}

func TestParseHora(t *testing.T) {
	validos := map[string]string{"07:44": "07:44", "7:44": "07:44", "00:00": "00:00", "23:59": "23:59"}
	for in, want := range validos {
		got, err := parseHora(in)
		if err != nil || fmtHora(got) != want {
			t.Errorf("parseHora(%q) = %s, %v; esperado %s", in, fmtHora(got), err, want)
		}
	}
	for _, in := range []string{"", "0744", "24:00", "12:60", "12:5", "ab:cd", "12:30:00"} {
		if _, err := parseHora(in); err == nil {
			t.Errorf("parseHora(%q) deveria falhar", in)
		}
	}
}

func TestParseMarcacoes(t *testing.T) {
	got, err := parseMarcacoes([]string{"07:44", "12:43 13:19"})
	if err != nil || len(got) != 3 {
		t.Fatalf("esperado 3 marcações, obtido %v, %v", got, err)
	}
	if _, err := parseMarcacoes([]string{"07:44", "07:00"}); err == nil {
		t.Error("marcações fora de ordem deveriam falhar")
	}
	if _, err := parseMarcacoes([]string{"07:44", "07:44"}); err == nil {
		t.Error("marcações repetidas deveriam falhar")
	}
}

func TestFmtDur(t *testing.T) {
	casos := map[time.Duration]string{
		0:                            "0min",
		36 * time.Minute:             "36min",
		8 * time.Hour:                "8h",
		4*time.Hour + 59*time.Minute: "4h59",
		1*time.Hour + 5*time.Minute:  "1h05",
	}
	for in, want := range casos {
		if got := fmtDur(in); got != want {
			t.Errorf("fmtDur(%v) = %s; esperado %s", in, got, want)
		}
	}
}

func TestCalcular(t *testing.T) {
	casos := []struct {
		nome             string
		marcacoes        []string
		agora            string
		saida            string
		trabalhadoAgora  string
		intervalo        string // total
		presumido, pausa bool
	}{
		{"exemplo com 3 marcações", []string{"07:44", "12:43", "13:19"}, "15:08", "16:20", "6h48", "36min", false, false},
		{"1 marcação presume almoço", []string{"07:44"}, "10:00", "16:14", "2h16", "30min", true, false},
		{"1 marcação após 12:00 não presume", []string{"12:30"}, "13:00", "20:30", "30min", "0min", false, false},
		{"em pausa considera retorno agora", []string{"07:44", "12:43"}, "13:00", "16:01", "4h59", "17min", false, true},
		{"5 marcações somam intervalos", []string{"07:44", "10:00", "10:15", "12:00", "12:40"}, "16:00", "16:39", "7h21", "55min", false, false},
		{"hora extra", []string{"07:44", "12:43", "13:00"}, "17:00", "16:01", "8h59", "17min", false, false},
		{"jornada completa antes da última saída", []string{"07:00", "15:30", "16:00"}, "16:10", "15:00", "8h40", "30min", false, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r := calcular(hs(c.marcacoes...), h(c.agora))
			var total time.Duration
			for _, iv := range r.intervalos {
				total += iv.fim - iv.inicio
			}
			if fmtHora(r.saida) != c.saida {
				t.Errorf("saída = %s; esperado %s", fmtHora(r.saida), c.saida)
			}
			if fmtDur(r.trabalhadoAgora) != c.trabalhadoAgora {
				t.Errorf("trabalhado agora = %s; esperado %s", fmtDur(r.trabalhadoAgora), c.trabalhadoAgora)
			}
			if fmtDur(total) != c.intervalo {
				t.Errorf("intervalo = %s; esperado %s", fmtDur(total), c.intervalo)
			}
			if r.presumido != c.presumido || r.emPausa != c.pausa {
				t.Errorf("presumido=%v pausa=%v; esperado %v %v", r.presumido, r.emPausa, c.presumido, c.pausa)
			}
		})
	}
}

func TestPersistencia(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	if _, err := carregarUltimas(); err == nil {
		t.Fatal("sem arquivo salvo deveria falhar")
	}
	if err := salvarUltimas(hs("07:44", "12:43", "13:19")); err != nil {
		t.Fatal(err)
	}
	got, err := carregarUltimas()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "07:44" || got[2] != "13:19" {
		t.Errorf("marcações carregadas = %v", got)
	}
}
