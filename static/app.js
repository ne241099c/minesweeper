// GoのWASMをロードするおまじない
const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    console.log("WASM Loaded");
    resetGame(); // ロード完了したらゲーム開始
});

function render(jsonStr) {
    const grid = JSON.parse(jsonStr);
    const board = document.getElementById('board');
    board.innerHTML = '';
    const status = document.getElementById('status');

    let gameOver = false;

    grid.forEach((row, y) => {
        row.forEach((cell, x) => {
            const div = document.createElement('div');
            div.className = 'cell';

            if (cell.state === 'opened') {
                div.classList.add('opened');
                if (cell.is_mine) {
                    div.classList.add('mine');
                    div.innerText = "💣";
                    gameOver = true;
                } else if (cell.count > 0) {
                    div.innerText = cell.count;
                    div.classList.add('n' + cell.count);
                }
            } else if (cell.state === 'flagged') {
                div.innerText = "🚩";
                // 右クリックでフラッグ解除できるように
                div.oncontextmenu = (e) => {
                    e.preventDefault();
                    toggleFlag(x, y);
                };
            } else {
                // 未開封
                div.onclick = () => openCell(x, y);
                // 右クリックでフラッグ
                div.oncontextmenu = (e) => {
                    e.preventDefault();
                    toggleFlag(x, y);
                };
            }
            board.appendChild(div);
        });
    });

    if (gameOver) {
        status.innerText = "GAME OVER!";
        status.style.color = "red";
    } else {
        status.innerText = "";
    }
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