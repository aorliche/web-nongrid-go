import {$, $$, drawText} from './util.js';
import {noFillFn, neverFillFn, Board} from './board.js';
import {Point, Edge, Polygon, randomEdgePoint} from './primitives.js';

let boardjson = null;

function initBoard(board) {
    if (!boardjson) {
        board.loop([(a,b) => board.fill(a,6,b)]);
        board.loop([(a,b) => board.fill(a,3,b)]);
        board.loop([(a,b) => board.fill(a,4,b)]);
        board.loop([(a,b) => board.fill(a,3,b)]);
        board.loop([(a,b) => board.fill(a,4,b)]);
        board.loop([(a,b) => board.fill(a,3,b)]);
        board.loop([noFillFn, (a,b) => board.fill(a,4,b)]);
        board.loop([(a,b) => board.placeOne(a,3,b)]);
        board.loop([(a,b) => board.fill(a,6,b)]);
        board.loop([(a,b) => board.fill(a,3,b)]);
        board.loop([(a,b) => board.fill(a,3,b)]);
        board.initNeighbors();
        board.repaint();
        return;
    }
    const boardplan = JSON.parse(boardjson);
    const fn = (typ, n) => {
        if (n == -1) {
            return neverFillFn;
        } else if (n == 0) {
            return noFillFn;
        } else if (typ == 'fill') {
            return (a, b) => board.fill(a,n,b);
        } else {
            return (a, b) => board.placeOne(a,n,b);
        }
    }
    boardplan.forEach(round => {
        const arr = [];
        for (let i=0; i<round.sav.length; i++) {
            arr.push(fn(round.typ, round.sav[i].n));
        }
        board.loop(arr);
    });
    board.initNeighbors();
    board.repaint();
}

function getLastMove(hist, cur) {
    if (hist.length == 0) return;
    const last = JSON.parse(hist[hist.length-1]);
    for (let i=0; i<last.length; i++) {
        if (last[i].player != cur[i].player && cur[i].player != null) {
            return last[i].id;
        }
    }
    return null;
}

function setupListeners(game) {
    game.conn.onmessage = e => {
        const json = JSON.parse(e.data);
        if (json.Action == "New") {
            game.id = json.Key;
            $('#vertices').innerText = game.board.points.length;
            $('#black').innerText = '';
            $('#white').innerText = '';
            return;
        }
        if (json.Action == "Join" || json.Action == "Move") {
            // Regenerate board from boardplan if needed
            if (json.Action == "Join") {
                if (json.BoardPlan) {
                    boardjson = json.BoardPlan;
                } else {
                    boardjson = null;
                }
                game.board = new Board($('#canvas'));
                initBoard(game.board);
                $('#vertices').innerText = game.board.points.length;
            }
            const pts = JSON.parse(json.Payload);
            game.board.lastId = getLastMove(game.board.history, pts);
            game.board.history.push(JSON.stringify(pts));
            game.board.loadPoints(pts);
            game.board.repaint();
            game.board.player = json.Player;
            game.passes = 0;
            const [bscore, wscore] = game.board.getScores();
            $('#black').innerText = bscore;
            $('#white').innerText = wscore;
            return;
        }
        if (json.Action == "Chat") {
            $('#chat').value += `${json.Player}: ${json.Payload}\n`;
            $('#chat').scrollTop = $('#chat').scrollHeight;
            return;
        }
        if (json.Action == "Pass") {
            game.board.player = json.Player;
            $('#chat').value += `${json.Payload}: has passed\n`;
            if (++game.passes >= 2) {
                $('#chat').value += `Two passes in a row. The game is over!\n`;
                const [bscore, wscore] = game.board.getScores();
                const canvas = $('#canvas');
                const ctx = canvas.getContext('2d');
                drawText(ctx, `Black: ${bscore}`, new Point(canvas.width/2, 300), 'red', 'bold 48px sans', true);
                drawText(ctx, `White: ${wscore}`, new Point(canvas.width/2, 350), 'red', 'bold 48px sans', true);
            }
            $('#chat').scrollTop = $('#chat').scrollHeight;
            return;
        }
        if (json.Action == "Concede") {
            $('#chat').value += `Concession!\n`;
            $('#chat').value += `${json.Payload} wins!\n`;
            $('#chat').scrollTop = $('#chat').scrollHeight;
            const ctx = canvas.getContext('2d');
            drawText(ctx, `${json.Payload} wins!`, new Point(canvas.width/2, 300), 'red', 'bold 48px sans', true);
            // Manually end the game
            game.passes = 2;
            return;
        }
    }
}   

window.addEventListener('load', () => {
    const canvas = $('#canvas');

    // Separate connection in game object for game
    // This connection only for listing games
    const conn = new WebSocket(`ws://${location.host}/list`);

    let player = null;
    let game = null;

    $('#new').addEventListener('click', () => {
        game = {board: new Board(canvas), player: 'black', passes: 0};
        initBoard(game.board); 
        const pts = game.board.savePoints();
        game.conn = new WebSocket(`ws://${location.host}/ws`);
        game.conn.onopen = () => {
            game.conn.send(JSON.stringify({Action: 'New', BoardPlan: boardjson ? boardjson : "", Payload: JSON.stringify(pts)}));
        };
        setupListeners(game);
    });

    $('#join').addEventListener('click', () => {
        const sel = $('select[name="games-list"]');    
        if (sel.selectedIndex == -1) return;
        const id = sel.options[sel.selectedIndex].value;
        if (game && game.id == id) return;
        game = {board: new Board(canvas), player: 'white', passes: 0};
        initBoard(game.board); 
        game.conn = new WebSocket(`ws://${location.host}/ws`);
        game.id = parseInt(id);
        game.conn.onopen = () => {
            game.conn.send(JSON.stringify({Action: 'Join', Key: game.id}));
        };
        setupListeners(game);
    });
    
    $('#canvas').addEventListener('mousemove', (e) => {
        if (!game || !game.board || game.player != game.board.player || game.passes >= 2) return;
        game.board.hover(e.offsetX, e.offsetY);
        game.board.repaint();
    });

    $('#canvas').addEventListener('click', (e) => {
        if (!game || !game.board || game.player != game.board.player || game.passes >= 2) return;
        const res = game.board.click(e.offsetX, e.offsetY);
        if (!res) return;
        game.board.repaint();
        game.conn.send(JSON.stringify({Action: 'Move', Key: game.id, Payload: JSON.stringify(game.board.savePoints())}));
        game.conn.send(JSON.stringify({Key: game.id, Action: 'Chat', Payload: `has moved`}));
    });

    function sendMessage() {
        if (!game || !game.conn) return;
        game.conn.send(JSON.stringify({Key: game.id, Action: 'Chat', Payload: $('#message').value}));
        $('#message').value = '';
    }

    $('#message').addEventListener('keyup', (e) => {
        if (e.key == 'Enter' || e.keyCode == 13) {
            sendMessage();
        }
    });

    $('#send').addEventListener('click', sendMessage);
    $('#pass').addEventListener('click', () => {
        if (!game || !game.conn) return;
        if (game.board.player != game.player) return;
        game.conn.send(JSON.stringify({Key: game.id, Action: 'Pass'}));
    });

    // This conn only used for listing games
    conn.onmessage = e => {
        const json = JSON.parse(e.data);

        // List of integer game ids
        json.sort((a,b) => a-b);

        const select = $('select[name="games-list"]');
        const toAdd = [];
        const games = [...select.options].map(opt => parseInt(opt.value));
        if (game && games.includes(game.id)) {
            for (let i=0; i<select.options.length; i++) {
                const opt = select.options[i];
                if (parseInt(opt.value) == game.id) {
                    select.remove(i);
                    break;
                }
            }
        } 
        for (let i=0; i<select.options.length; i++) {
            const opt = select.options[i];
            if (!json.includes(parseInt(opt.value))) {
                select.remove(i--);
            }
        }
        json.forEach(key => {
            if (!games.includes(key) && !(game && game.id == key)) {
                const opt = document.createElement('option');
                opt.value = key;
                opt.innerHTML = `Game ${key}`;
                select.appendChild(opt);
            }
        });
    }
    
    const connBoards = new WebSocket(`ws://${location.host}/boards`);

    setInterval(e => {
        if (!conn.readyState == 1) return;
        conn.send(JSON.stringify({Action: 'List'}));
    }, 1000);
    
    setInterval(e => {
        if (!connBoards.readyState == 1) return;
        connBoards.send(JSON.stringify({Action: 'List'}));
    }, 1000);

    $('#load').addEventListener('click', () => {
        const idx = $('#boards').selectedIndex;
        if (idx == -1) return;
        const req = {Action: 'Load', Player: $('#boards').options[idx].innerText};
        connBoards.send(JSON.stringify(req));
    });

    connBoards.onmessage = e => {
        const msg = JSON.parse(e.data);
        if (msg.Action == 'List') {
            JSON.parse(msg.Payload).forEach(name => {
                const existing = $$('#boards option');
                let found = false;
                for (let i=0; i<existing.length; i++) {
                    if (existing[i].innerText == name) {
                        found = true;
                        break;
                    }
                }
                if (!found) {
                    const opt = document.createElement('option');
                    opt.innerText = name;
                    $('#boards').appendChild(opt);
                }
            });
        } else if (msg.Action == 'Load') {
            boardjson = msg.Payload;
            initBoard(new Board(canvas));
        }
    }

    $('#concede').addEventListener('click', () => {
        if (!game || !game.conn) return;
        game.conn.send(JSON.stringify({Key: game.id, Action: 'Concede'}));
    });
});
