const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    console.log("WASM Loaded");
    // 初回はBotを動かさずに初期化
    resetGame(false);
});

// 設定取得
function getSettings() {
    const w = parseInt(document.getElementById('width').value) || 10;
    const h = parseInt(document.getElementById('height').value) || 10;
    const m = parseInt(document.getElementById('mines').value) || 10;
    // ランダムオープンのチェックボックス
    const autoOpen = document.getElementById('auto-open').checked;
    return { w, h, m, autoOpen };
}

const botLoopState = {
    intervalId: null,
    isRunning: false,
    currentRun: 0,
    maxRuns: 0,
    wins: 0,
    isBotReset: false
};

// ゲームリセット（最重要修正箇所）
function resetGame(isBotReset = false) {
    // 人間がボタンを押した場合、Botループを強制停止
    if (!isBotReset) {
        stopBotLoop();
    }

    if (typeof goNewGame === 'function') {
        const { w, h, m, autoOpen } = getSettings();
        // Botによるリセットの場合は「ランダムオープン」はBotの裁量に任せる（今回は設定に従う）
        const jsonStr = goNewGame(w, h, m, autoOpen);
        render(jsonStr);
    }
}

// Botループ開始（画面更新あり）
function startBotLoop() {
    if (botLoopState.isRunning) return;

    const runs = parseInt(document.getElementById('bot-runs').value) || 1;
    botLoopState.maxRuns = runs;
    botLoopState.currentRun = 0;
    botLoopState.wins = 0;
    botLoopState.isRunning = true;
    
    // Bot開始時は強制的にリセットしてスタート
    resetGame(true);
    runBotInterval();
}

function stopBotLoop() {
    if (botLoopState.intervalId) clearInterval(botLoopState.intervalId);
    botLoopState.isRunning = false;
    botLoopState.intervalId = null;
}

function runBotInterval() {
    botLoopState.intervalId = setInterval(() => {
        if (!botLoopState.isRunning) {
            stopBotLoop();
            return;
        }

        if (typeof goBotStep === 'function') {
            const jsonStr = goBotStep();
            let state = {};
            try { state = JSON.parse(jsonStr || "{}"); } catch(e){}
            
            render(jsonStr);

            if (state.is_game_over || state.is_game_clear) {
                clearInterval(botLoopState.intervalId);
                
                if (state.is_game_clear) botLoopState.wins++;
                botLoopState.currentRun++;
                
                updateStatus(`Game ${botLoopState.currentRun}/${botLoopState.maxRuns} (Wins: ${botLoopState.wins})`);

                if (botLoopState.currentRun < botLoopState.maxRuns) {
                    // 0.5秒待って次へ
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
    }, 50); // 速度調整
}

// ベンチマーク実行（画面更新なし・超高速）
function runBenchmark() {
    stopBotLoop(); // 通常ループは止める
    const { w, h, m } = getSettings();
    const runs = parseInt(document.getElementById('bot-runs').value) || 100;
    
    updateStatus("Running benchmark... please wait.");
    
    // UIが固まらないように少し待ってから実行
    setTimeout(() => {
        if (typeof goRunBenchmark === 'function') {
            const result = goRunBenchmark(w, h, m, runs);
            alert(result); // 結果をアラートまたはログに出す
            updateStatus("Benchmark finished.");
        }
    }, 100);
}

// 表示系ヘルパー
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

    // 残り地雷数
    const mineEl = document.getElementById('mine-count');
    if (mineEl) mineEl.innerText = gameState.mines_remaining;

    // ステータス（Bot実行中は上書きしない）
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