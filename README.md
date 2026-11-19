# 🌙 BrightControl

App para **macOS, Windows e Linux** que **apaga a tela quando a bateria está quase no fim** (e o carregador não está conectado), para o Mac durar mais alguns minutos — com um ícone lá em cima para ligar/desligar na hora, sem abrir Ajustes do Sistema.

Quando a bateria fica baixa (20% ou menos), ele também mostra **um ícone de bateria igual ao nativo do Mac**, em vermelho, com a porcentagem. Esse ícone é só indicador: clicar nele não faz nada.

![Barra de menus](docs/menubar.png)

<img src="docs/menu.png" width="360" alt="Menu do BrightControl">

## Instalar

### macOS

Cole no Terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/julioflima/bright-control/main/install.sh | bash
```

![Demo da instalação](docs/install-demo.gif)

O instalador:

1. baixa o último release e coloca `BrightControl.app` em `/Applications`;
2. remove o antigo `bright_control.sh` (LaunchAgent `com.usuario.controlbrillo`), se existir;
3. configura para abrir ao iniciar sessão e já abre o app — procure a 🌙 na barra de menus.

### Windows (10/11)

Cole no PowerShell:

```powershell
irm https://raw.githubusercontent.com/julioflima/bright-control/main/install.ps1 | iex
```

Instala em `%LOCALAPPDATA%\BrightControl` e abre o app; a 🌙 aparece na bandeja, perto do relógio (se não aparecer, clique na setinha `^` e arraste-a para a barra).

### Linux

```bash
curl -fsSL https://raw.githubusercontent.com/julioflima/bright-control/main/install.sh | bash
```

Instala em `~/.local/bin/brightcontrol`. Precisa de uma bandeja compatível (KDE, XFCE, Cinnamon, MATE; no GNOME, a extensão *AppIndicator*). Para apagar a tela usa o que existir: `xset` (X11), `kscreen-doctor` (KDE Wayland), GNOME Wayland via D-Bus ou `wlopm`.

> O app sempre abre ao iniciar a sessão, nos três sistemas.

> **MacBook com notch e barra cheia?** Ícones novos entram pela esquerda e podem ficar escondidos atrás do notch. O BrightControl já se posiciona perto do relógio; para mudar de lugar, segure **⌘** e arraste o ícone.

## O que tem no menu

| Opção | O que faz |
|---|---|
| **Apagar tela com bateria baixa** | Liga/desliga o comportamento (🌙 cheia = ligado, 🌙 vazada = desligado) |
| **Apagar quando chegar a…** | Limite: 2, 3, 5 (padrão), 10, 15 ou 20 % |
| **Pausar por…** | Desliga por 15 min, 30 min, 1 h ou 2 h e volta sozinho |
| **Ícone de bateria baixa na barra** | Mostra/esconde o ícone vermelho de bateria (aparece com 20% ou menos) |

Com a tela apagada, basta mexer no mouse/teclado: o app espera 20 s antes de apagar de novo, tempo de sobra para clicar na 🌙 e desligar ou pausar.

## Desinstalar

macOS e Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/julioflima/bright-control/main/uninstall.sh | bash
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/julioflima/bright-control/main/uninstall.ps1 | iex
```

## Compilar do código

| Sistema | Código | Build |
|---|---|---|
| macOS | `Sources/main.swift` (Swift/AppKit nativo) | `./build.sh` |
| Windows e Linux | `cross/` (Go + [fyne.io/systray](https://github.com/fyne-io/systray)) | `cross/build.sh` (compila os dois de qualquer sistema, sem cgo) |

No macOS, precisa só das Command Line Tools (`xcode-select --install`):

```bash
./build.sh                         # gera build/BrightControl.app e dist/BrightControl.zip
LOCAL_ZIP=dist/BrightControl.zip ./install.sh   # instala a versão local
```

Um arquivo só (`Sources/main.swift`), sem dependências. Bateria lida via IOKit; a tela é apagada com `pmset displaysleepnow` (não precisa de senha de administrador). Funciona no macOS 12+ em Apple Silicon e Intel.

Para tirar prints sem esperar a bateria acabar: `BrightControl --preview-low 8` (no Mac, `--open-menu` também abre o menu). No Windows/Linux, `brightcontrol --status` mostra a leitura da bateria e `--display-off` testa o apagar da tela.

## Licença

MIT
