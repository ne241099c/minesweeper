// WASMロード部分は変更なし
const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    console.log("WASM Loaded");
    resetGame();
});

// ゲーム設定を取得するヘルパー
function getSettings() {
    const w = parseInt(document.getElementById('width').value) || 10;
    const h = parseInt(document.getElementById('height').value) || 10;
    const m = parseInt(document.getElementById('mines').value) || 10;
    return { w, h, m };
}

function resetGame() {
    // 自動実行中なら一旦止める（安全策）
    if (botLoopState.isRunning && !botLoopState.isLoopingReset) {
        stopBotLoop();
    }

    if (typeof goNewGame === 'function') {
        const { w, h, m } = getSettings();
        // Go側に設定値を渡す
        const jsonStr = goNewGame(w, h, m);
        render(jsonStr);
    }
}

const botLoopState = {
    intervalId: null,
    isRunning: false,
    currentRun: 0,
    maxRuns: 0,
    wins: 0,
    losses: 0,
    isLoopingReset: false // ループ内でのリセット中かどうか
};

function startBotLoop() {
    if (botLoopState.isRunning) return;

    const runs = parseInt(document.getElementById('bot-runs').value) || 1;
    botLoopState.maxRuns = runs;
    botLoopState.currentRun = 0;
    botLoopState.wins = 0;
    botLoopState.losses = 0;
    botLoopState.isRunning = true;
    botLoopState.isLoopingReset = false;

    console.log(`Starting Bot Loop: ${runs} games`);
    
    // 最初のゲームを開始
    resetGame();
    runBotInterval();
}

function stopBotLoop() {
    if (botLoopState.intervalId) {
        clearInterval(botLoopState.intervalId);
    }
    botLoopState.isRunning = false;
    botLoopState.intervalId = null;
    console.log("Bot Loop Stopped");
}

function runBotInterval() {
    // 0.05秒ごとにBotを動かす
    botLoopState.intervalId = setInterval(() => {
        if (!botLoopState.isRunning) return;

        if (typeof goBotStep === 'function') {
            const jsonStr = goBotStep();
            // ゲーム結果判定のためにパース
            let state = {};
            try { state = JSON.parse(jsonStr || "{}"); } catch(e){}

            render(jsonStr);

            // ゲーム終了判定
            if (state.is_game_over || state.is_game_clear) {
                // インターバルを止める
                clearInterval(botLoopState.intervalId);
                
                // 結果記録
                if (state.is_game_clear) botLoopState.wins++;
                else botLoopState.losses++;
                
                botLoopState.currentRun++;
                
                updateStatus(`Game ${botLoopState.currentRun}/${botLoopState.maxRuns} Finished. (Wins: ${botLoopState.wins})`);

                // 次のゲームへ進むか終了か
                if (botLoopState.currentRun < botLoopState.maxRuns) {
                    // 少し待ってから次のゲームへ
                    setTimeout(() => {
                        if (!botLoopState.isRunning) return;
                        botLoopState.isLoopingReset = true;
                        resetGame(); // 次のゲーム開始
                        botLoopState.isLoopingReset = false;
                        runBotInterval(); // Bot再開
                    }, 1000); // 1秒ウェイト
                } else {
                    stopBotLoop();
                    updateStatus(`Finished! Wins: ${botLoopState.wins}, Losses: ${botLoopState.losses}`);
                }
            }
        }
    }, 50); 
}

function updateStatus(msg) {
    const status = document.getElementById('status');
    if (status) status.innerText = msg;
}

function render(jsonStr) {
    if (!jsonStr || jsonStr === "{}") return;
    let gameState;
    try { gameState = JSON.parse(jsonStr); } catch (e) { return; }
    
    const board = document.getElementById('board');
    // 盤面サイズが変わったときに再生成
    const currentW = gameState.cells[0].length;
    const currentH = gameState.cells.length;
    const boardStyleW = currentW * 32; // cell width + gap
    
    // boardのスタイルを動的に調整（折り返しを防ぐため）
    board.style.width = `${boardStyleW}px`;
    board.style.gridTemplateColumns = `repeat(${currentW}, 30px)`;

    // 初回生成 or サイズ変更時
    if (board.childElementCount !== currentH * currentW) {
        board.innerHTML = '';
        gameState.cells.forEach((row, y) => {
            row.forEach((cell, x) => {
                const div = document.createElement('div');
                div.id = `cell-${x}-${y}`;
                div.className = 'cell';
                div.onclick = () => openCell(x, y);
                div.oncontextmenu = (e) => { e.preventDefault(); toggleFlag(x, y); };
                board.appendChild(div);
            });
        });
    }

    // 差分更新
    const mineCountSpan = document.getElementById('mine-count');
    if (mineCountSpan) mineCountSpan.innerText = gameState.mines_remaining;

    // ゲーム終了時のメッセージ
    const status = document.getElementById('status');
    if (!botLoopState.isRunning) {
        if (gameState.is_game_over) {
            status.innerText = "GAME OVER!";
            status.style.color = "red";
        } else if (gameState.is_game_clear) {
            status.innerText = "GAME CLEAR!! 🎉";
            status.style.color = "lime";
        } else {
            status.innerText = "";
        }
    }

    const isFinished = gameState.is_game_over || gameState.is_game_clear;

    gameState.cells.forEach((row, y) => {
        row.forEach((cellData, x) => {
            const div = document.getElementById(`cell-${x}-${y}`);
            if (!div) return;
            
            // クラスのリセット
            div.className = 'cell';
            div.innerText = '';
            
            if (cellData.state === 'opened') {
                div.classList.add('opened');
                if (cellData.is_mine) {
                    div.classList.add('mine');
                    div.innerText = "💣";
                } else if (cellData.count > 0) {
                    div.innerText = cellData.count;
                    div.classList.add('n' + cellData.count);
                }
            } else if (cellData.state === 'flagged') {
                div.innerText = "🚩";
            }
        });
    });
}

// ラッパー関数
function openCell(x, y) {
    if (typeof goOpenCell === 'function') render(goOpenCell(x, y));
}
function toggleFlag(x, y) {
    if (typeof goToggleFlag === 'function') render(goToggleFlag(x, y));
}

// Bot関数
function runBotStep() {
    if (typeof goBotStep === 'function') {
        const jsonStr = goBotStep();
        render(jsonStr);
    }
}

function toggleAutoBot() {
    if (botLoopState.isRunning) {
        stopBotLoop();
    } else {
        startBotLoop();
    }
}