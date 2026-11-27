package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	jornada         = 8 * time.Hour
	intervaloMinimo = 30 * time.Minute
)

// Almoço presumido quando só existe 1 marcação
var (
	almocoPresumidoInicio = hm(12, 0)
	almocoPresumidoFim    = hm(12, 30)
)

type bloco struct {
	inicio, fim time.Duration // minutos desde 00:00
	aberto      bool          // true = sem marcação de fim (ainda em andamento)
}

func main() {
	agoraFlag := flag.String("agora", "", "simula o horário atual (HH:MM), útil para testes")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "uso: ponto [-agora HH:MM] HH:MM [HH:MM ...]")
		fmt.Fprintln(os.Stderr, "exemplo: ponto 07:44 12:43 13:19")
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	marcacoes, err := parseMarcacoes(flag.Args())
	if err != nil {
		erro(err.Error())
	}

	agora := agoraAtual()
	if *agoraFlag != "" {
		agora, err = parseHora(*agoraFlag)
		if err != nil {
			erro("valor inválido em -agora: " + err.Error())
		}
	}

	ultima := marcacoes[len(marcacoes)-1]
	if agora < ultima {
		erro(fmt.Sprintf("a última marcação (%s) é posterior ao horário atual (%s)", fmtHora(ultima), fmtHora(agora)))
	}

	var avisos []string
	calc := append([]time.Duration{}, marcacoes...) // marcações usadas no cálculo
	emPausa := len(marcacoes)%2 == 0
	presumido := false

	switch {
	case len(marcacoes) == 1 && marcacoes[0] < almocoPresumidoInicio:
		calc = []time.Duration{marcacoes[0], almocoPresumidoInicio, almocoPresumidoFim}
		presumido = true
		avisos = append(avisos, fmt.Sprintf("Almoço ainda não marcado: cálculo considera intervalo de %s a %s.",
			fmtHora(almocoPresumidoInicio), fmtHora(almocoPresumidoFim)))
	case emPausa:
		calc = append(calc, agora)
		avisos = append(avisos, fmt.Sprintf("Em pausa desde %s: cálculo considera que você volta agora (%s).",
			fmtHora(ultima), fmtHora(agora)))
	}

	blocos, intervalos := montarBlocos(calc)

	// Trabalhado até agora: sempre com base nas marcações REAIS
	trabalhadoAgora := trabalhadoAte(marcacoes, agora)

	// Horário de saída: percorre os blocos até completar a jornada
	saida := calcularSaida(blocos)

	// ---- Saída ----
	if len(marcacoes) > 1 {
		linha("Trabalhado até "+fmtHora(ultima), fmtDur(trabalhadoAte(marcacoes, ultima)))
	}
	linha("Trabalhado até agora", fmtDur(trabalhadoAgora))

	var totalIntervalo time.Duration
	var detalhes []string
	for _, iv := range intervalos {
		totalIntervalo += iv.fim - iv.inicio
		detalhes = append(detalhes, fmtHora(iv.inicio)+"–"+fmtHora(iv.fim))
	}
	if len(intervalos) > 0 {
		txt := fmt.Sprintf("%s (%s)", fmtDur(totalIntervalo), strings.Join(detalhes, ", "))
		if presumido {
			txt += " [presumido]"
		}
		if totalIntervalo < intervaloMinimo {
			txt += fmt.Sprintf(" (abaixo do mínimo de %s)", fmtDur(intervaloMinimo))
		}
		rotulo := "Intervalo"
		if len(intervalos) > 1 {
			rotulo = "Intervalos"
		}
		linha(rotulo, txt)
	} else {
		linha("Intervalo", "nenhum")
	}

	linha("Saída prevista", fmtHora(saida))
	if falta := saida - agora; falta > 0 {
		linha("Faltam", fmt.Sprintf("%s  (agora %s)", fmtDur(falta), fmtHora(agora)))
	} else {
		linha("Hora extra", fmt.Sprintf("%s  (agora %s)", fmtDur(-falta), fmtHora(agora)))
	}

	for _, a := range avisos {
		fmt.Println("* " + a)
	}
}

// montarBlocos separa as marcações em blocos de trabalho (E→S) e intervalos (S→E).
// Com quantidade ímpar, o último bloco fica aberto.
func montarBlocos(m []time.Duration) (trabalho, intervalos []bloco) {
	for i := 0; i < len(m); i += 2 {
		if i+1 < len(m) {
			trabalho = append(trabalho, bloco{inicio: m[i], fim: m[i+1]})
		} else {
			trabalho = append(trabalho, bloco{inicio: m[i], aberto: true})
		}
	}
	for i := 1; i+1 < len(m); i += 2 {
		intervalos = append(intervalos, bloco{inicio: m[i], fim: m[i+1]})
	}
	return
}

func calcularSaida(blocos []bloco) time.Duration {
	var acumulado time.Duration
	for _, b := range blocos {
		if b.aberto {
			return b.inicio + (jornada - acumulado)
		}
		dur := b.fim - b.inicio
		if acumulado+dur >= jornada {
			return b.inicio + (jornada - acumulado)
		}
		acumulado += dur
	}
	// Não deve acontecer (sempre há um bloco aberto no cálculo)
	return blocos[len(blocos)-1].fim
}

// trabalhadoAte soma o tempo trabalhado pelas marcações reais até o instante t.
func trabalhadoAte(m []time.Duration, t time.Duration) time.Duration {
	var total time.Duration
	for i := 0; i < len(m); i += 2 {
		inicio := m[i]
		fim := t
		if i+1 < len(m) && m[i+1] < t {
			fim = m[i+1]
		}
		if fim > inicio {
			total += fim - inicio
		}
	}
	return total
}

func parseMarcacoes(args []string) ([]time.Duration, error) {
	var out []time.Duration
	for _, a := range args {
		for _, p := range strings.Fields(a) { // aceita também "07:44 12:43" entre aspas
			h, err := parseHora(p)
			if err != nil {
				return nil, err
			}
			if len(out) > 0 && h <= out[len(out)-1] {
				return nil, fmt.Errorf("marcações fora de ordem: %s vem depois de %s", fmtHora(h), fmtHora(out[len(out)-1]))
			}
			out = append(out, h)
		}
	}
	return out, nil
}

func parseHora(s string) (time.Duration, error) {
	partes := strings.Split(s, ":")
	if len(partes) != 2 {
		return 0, fmt.Errorf("horário inválido %q (use HH:MM)", s)
	}
	h, err1 := strconv.Atoi(partes[0])
	m, err2 := strconv.Atoi(partes[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 || len(partes[1]) != 2 {
		return 0, fmt.Errorf("horário inválido %q (use HH:MM)", s)
	}
	return hm(h, m), nil
}

func agoraAtual() time.Duration {
	n := time.Now()
	return hm(n.Hour(), n.Minute())
}

func hm(h, m int) time.Duration {
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
}

func fmtHora(d time.Duration) string {
	min := int(d.Minutes())
	return fmt.Sprintf("%02d:%02d", min/60, min%60)
}

func fmtDur(d time.Duration) string {
	min := int(d.Minutes())
	h, m := min/60, min%60
	switch {
	case h == 0:
		return fmt.Sprintf("%dmin", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dh%02d", h, m)
	}
}

func linha(rotulo, valor string) {
	pontos := 28 - len([]rune(rotulo))
	if pontos < 3 {
		pontos = 3
	}
	fmt.Printf("%s %s %s\n", rotulo, strings.Repeat(".", pontos), valor)
}

func erro(msg string) {
	fmt.Fprintln(os.Stderr, "erro: "+msg)
	os.Exit(1)
}
