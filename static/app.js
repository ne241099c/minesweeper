// GoのWASMをロードするおまじない
const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    console.log("WASM Loaded");
    resetGame(); // ロード完了したらゲーム開始
});

function render(jsonStr) {
    if (!jsonStr || jsonStr === "{}") {
        console.warn("Received empty state");
        return;
    }

    let gameState;
    try {
        gameState = JSON.parse(jsonStr);
    } catch (e) {
        console.error("JSON Parse Error:", e, jsonStr);
        return;
    }
    
    const grid = gameState.cells;
    if (!grid) return; // データがない場合は終了

    const board = document.getElementById('board');
    board.innerHTML = '';
    
    const status = document.getElementById('status');
    const mineCountSpan = document.getElementById('mine-count');

    if (mineCountSpan) mineCountSpan.innerText = gameState.mines_remaining;

    if (gameState.is_game_over) {
        status.innerText = "GAME OVER!";
        status.style.color = "red";
    } else if (gameState.is_game_clear) {
        status.innerText = "GAME CLEAR!! 🎉";
        status.style.color = "lime";
    } else {
        status.innerText = "";
    }

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
                if (!isFinished) {
                    div.oncontextmenu = (e) => {
                        e.preventDefault();
                        toggleFlag(x, y);
                    };
                }
            } else {
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

function openCell(x, y) {
    console.log(`Click: ${x}, ${y}`); // デバッグログ
    if (typeof goOpenCell === 'function') {
        const jsonStr = goOpenCell(x, y);
        render(jsonStr);
    }
}

function toggleFlag(x, y) {
    if (typeof goToggleFlag === 'function') {
        const jsonStr = goToggleFlag(x, y);
        render(jsonStr);
    }
}

// 1手だけBotを動かす
function runBotStep() {
    if (typeof goBotStep === 'function') {
        const jsonStr = goBotStep();
        render(jsonStr);
    }
}

// 自動再生用
let autoBotInterval = null;

function toggleAutoBot() {
    if (autoBotInterval) {
        // 停止
        clearInterval(autoBotInterval);
        autoBotInterval = null;
        console.log("Auto Bot Stopped");
    } else {
        // 開始（0.1秒ごとに実行）
        console.log("Auto Bot Started");
        autoBotInterval = setInterval(() => {
            if (typeof goBotStep === 'function') {
                const jsonStr = goBotStep();
                
                // ゲーム終了判定をして止める
                const state = JSON.parse(jsonStr || "{}");
                if (state.is_game_over || state.is_game_clear) {
                    clearInterval(autoBotInterval);
                    autoBotInterval = null;
                }
                
                render(jsonStr);
            }
        }, 100);
    }
}

function resetGame() {
    if (autoBotInterval) {
        clearInterval(autoBotInterval);
        autoBotInterval = null;
    }
    if (typeof goNewGame === 'function') {
        const jsonStr = goNewGame();
        render(jsonStr);
    }
}