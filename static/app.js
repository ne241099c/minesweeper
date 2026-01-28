const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    console.log("WASM Loaded");
    resetGame(false);
});

function getSettings() {
    const w = parseInt(document.getElementById('width').value) || 10;
    const h = parseInt(document.getElementById('height').value) || 10;
    const m = parseInt(document.getElementById('mines').value) || 10;
    return { w, h, m };
}

const botLoopState = {
    intervalId: null,
    isRunning: false,
    currentRun: 0,
    maxRuns: 0,
    wins: 0,
    isBotReset: false,
    isPaused: false // ポーズフラグ
};

function resetGame(isBotReset = false) {
    if (!isBotReset) {
        stopBotLoop();
    }

    if (typeof goNewGame === 'function') {
        const { w, h, m } = getSettings();
        const jsonStr = goNewGame(w, h, m);
        render(jsonStr);
    }
}

function startBotLoop() {
    if (botLoopState.isRunning) return;

    const runs = parseInt(document.getElementById('bot-runs').value) || 1;
    botLoopState.maxRuns = runs;
    botLoopState.currentRun = 0;
    botLoopState.wins = 0;
    botLoopState.isRunning = true;
    botLoopState.isPaused = false; // 開始時はポーズ解除
    
    // ボタンの見た目を更新
    updatePauseButton();
    
    resetGame(true);
    runBotInterval();
}

// ポーズボタンの切り替え
function toggleBotLoop() {
    if (!botLoopState.isRunning) return;

    botLoopState.isPaused = !botLoopState.isPaused;
    updatePauseButton();
}

function updatePauseButton() {
    const btn = document.getElementById('bot-pause-btn');
    if (btn) {
        if (botLoopState.isPaused) {
            btn.innerText = "▶ Resume";
            btn.style.backgroundColor = "#4CAF50"; // 緑色（再生）
        } else {
            btn.innerText = "⏸ Pause";
            btn.style.backgroundColor = "#f44336"; // 赤色（停止っぽい色）
        }
    }
}

function stopBotLoop() {
    if (botLoopState.intervalId) clearInterval(botLoopState.intervalId);
    botLoopState.isRunning = false;
    botLoopState.intervalId = null;
    botLoopState.isPaused = false;
    // ボタンをPauseに戻しておく（次回用）
    updatePauseButton();
}

function runBotInterval() {
    botLoopState.intervalId = setInterval(() => {
        if (!botLoopState.isRunning) {
            stopBotLoop();
            return;
        }

        // ポーズ中は処理をスキップ
        if (botLoopState.isPaused) {
            return;
        }

        if (typeof goBotStep === 'function') {
            const jsonStr = goBotStep();
            let state = {};
            try { state = JSON.parse(jsonStr || "{}"); } catch(e){}
            
            render(jsonStr);

            if (state.is_game_over || state.is_game_clear) {
                // ここではインターバルを止めず、次へ進む処理をする
                // ただし、もしここでポーズさせたいなら考慮が必要だが、
                // 今回は「ゲーム終了→次へ」の流れは止まらないものとする
                
                clearInterval(botLoopState.intervalId);
                
                if (state.is_game_clear) botLoopState.wins++;
                botLoopState.currentRun++;
                
                updateStatus(`Game ${botLoopState.currentRun}/${botLoopState.maxRuns} (Wins: ${botLoopState.wins})`);

                if (botLoopState.currentRun < botLoopState.maxRuns) {
                    setTimeout(() => {
                        if (!botLoopState.isRunning) return;
                        resetGame(true);
                        runBotInterval();
                    }, 500);
                } else {
                    stopBotLoop();
                    updateStatus(`Finished! Win Rate: ${((botLoopState.wins/botLoopState.maxRuns)*100).toFixed(1)}%`);
                }
            }
        }
    }, 50);
}

// ベンチマーク実行
function runBenchmark() {
    stopBotLoop();
    const { w, h, m } = getSettings();
    const runs = parseInt(document.getElementById('bot-runs').value) || 100;
    
    updateStatus("Running benchmark... please wait.");
    
    setTimeout(() => {
        if (typeof goRunBenchmark === 'function') {
            const result = goRunBenchmark(w, h, m, runs);
            alert(result);
            updateStatus("Benchmark finished.");
        }
    }, 100);
}

function updateStatus(msg) {
    const el = document.getElementById('status');
    if (el) el.innerText = msg;
}

function render(jsonStr) {
    if (!jsonStr || jsonStr === "{}") return;
    let gameState;
    try { gameState = JSON.parse(jsonStr); } catch(e) { return; }
    
    const board = document.getElementById('board');
    const w = gameState.cells[0].length;
    board.style.width = `${w * 32}px`;
    board.style.gridTemplateColumns = `repeat(${w}, 30px)`;
    
    if (board.childElementCount !== gameState.cells.length * w) {
        board.innerHTML = '';
        gameState.cells.forEach((row, y) => {
            row.forEach((_, x) => {
                const div = document.createElement('div');
                div.id = `c-${x}-${y}`;
                div.className = 'cell';
                div.onclick = () => openCell(x, y);
                div.oncontextmenu = (e) => { e.preventDefault(); toggleFlag(x, y); };
                board.appendChild(div);
            });
        });
    }

    const mineEl = document.getElementById('mine-count');
    if (mineEl) mineEl.innerText = gameState.mines_remaining;

    if (!botLoopState.isRunning) {
        if (gameState.is_game_over) updateStatus("GAME OVER");
        else if (gameState.is_game_clear) updateStatus("CLEARED!");
        else updateStatus("");
    }

    gameState.cells.forEach((row, y) => {
        row.forEach((c, x) => {
            const div = document.getElementById(`c-${x}-${y}`);
            if(!div) return;
            div.className = 'cell';
            div.innerText = '';
            if (c.state === 'opened') {
                div.classList.add('opened');
                if (c.is_mine) { div.classList.add('mine'); div.innerText = "💣"; }
                else if (c.count > 0) { div.classList.add('n'+c.count); div.innerText = c.count; }
            } else if (c.state === 'flagged') {
                div.innerText = "🚩";
            }
        });
    });
}

function openCell(x, y) { if(typeof goOpenCell === 'function') render(goOpenCell(x, y)); }
function toggleFlag(x, y) { if(typeof goToggleFlag === 'function') render(goToggleFlag(x, y)); }