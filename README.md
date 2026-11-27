# ponto

CLI para calcular o horário de saída do trabalho a partir das marcações de ponto do dia.

## Instalação

Requer Go instalado:

```bash
go install github.com/jean-with-java/ponto@latest
```

## Uso

Informe as marcações do dia em ordem (entrada, saída, entrada...):

```bash
ponto 07:44 12:43 13:19
```

```
Trabalhado até 13:19 ........ 4h59
Trabalhado até agora ........ 6h48
Intervalo ................... 36min (12:43–13:19)
Saída prevista .............. 16:20
Faltam ...................... 1h12  (agora 15:08)
```

Sem argumentos, reaproveita as últimas marcações válidas do dia:

```bash
ponto
```

## Regras

- Jornada de 8h.
- Intervalo mínimo de 30min (apenas informativo).
- Com 1 marcação, considera almoço das 12:00 às 12:30.
- Com número par de marcações (em pausa), considera retorno no horário atual.
- `-agora HH:MM` simula o horário atual.
