// GoのWASMをロードするおまじない
const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    console.log("WASM Loaded");
    resetGame(); // ロード完了したらゲーム開始
});

function render(jsonStr) {
    const gameState = JSON.parse(jsonStr);
    const grid = gameState.cells;
    
    const board = document.getElementById('board');
    board.innerHTML = '';
    
    const status = document.getElementById('status');
    const mineCountSpan = document.getElementById('mine-count');

    // 残り地雷数の更新
    mineCountSpan.innerText = gameState.mines_remaining;

    // ステータス表示
    if (gameState.is_game_over) {
        status.innerText = "GAME OVER!";
        status.style.color = "red";
    } else if (gameState.is_game_clear) {
        status.innerText = "GAME CLEAR!! 🎉";
        status.style.color = "lime";
    } else {
        status.innerText = "";
    }

    // ゲーム終了時はクリックできないようにするフラグ
    const isFinished = gameState.is_game_over || gameState.is_game_clear;

    grid.forEach((row, y) => {
        row.forEach((cell, x) => {
            const div = document.createElement('div');
            div.className = 'cell';

            if (cell.state === 'opened') {
                div.classList.add('opened');
                if (cell.is_mine) {
                    div.classList.add('mine');
                    div.innerText = "💣";
                } else if (cell.count > 0) {
                    div.innerText = cell.count;
                    div.classList.add('n' + cell.count);
                }
            } else if (cell.state === 'flagged') {
                div.innerText = "🚩";
                // ゲーム中でなければ右クリック解除可能
                if (!isFinished) {
                    div.oncontextmenu = (e) => {
                        e.preventDefault();
                        toggleFlag(x, y);
                    };
                }
            } else {
                // 未開封
                if (!isFinished) {
                    div.onclick = () => openCell(x, y);
                    div.oncontextmenu = (e) => {
                        e.preventDefault();
                        toggleFlag(x, y);
                    };
                }
            }
            board.appendChild(div);
        });
    });
}

// Goの関数を呼び出すラッパー
function openCell(x, y) {
    const jsonStr = goOpenCell(x, y); // Goの関数を直接実行！
    render(jsonStr);
}

function toggleFlag(x, y) {
    const jsonStr = goToggleFlag(x, y); // Goの関数を直接実行！
    render(jsonStr);
}

function resetGame() {
    if (typeof goNewGame === 'function') {
        const jsonStr = goNewGame();
        render(jsonStr);
    }
}