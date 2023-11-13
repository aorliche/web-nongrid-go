
import {$, $$} from './util.js';
import {noFillFn, Board} from './board.js';
import {Point, Edge, Polygon, randomEdgePoint} from './primitives.js';

function initBoard(board) {
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
}

window.addEventListener('load', () => {
    const canvas = $('#canvas');
    const board = new Board(canvas);

    const conn = new WebSocket(`ws://${location.host}/ws`);
    let player = null;
    let game = null;

    $('#new').addEventListener('click', () => {
        player = 'black';
        initBoard(board); 
        const pts = board.savePoints();
        conn.send(JSON.stringify({Action: 'New', Payload: JSON.stringify(pts)}));
    });

    $('#join').addEventListener('click', () => {
        player = 'white';
        initBoard(board); 
        const sel = $('select[name="games-list"]');    
        const id = sel.options[sel.selectedIndex].value;
        game = {id: parseInt(id)};
        conn.send(JSON.stringify({Action: 'Join', Key: game.id}));
    });
    
    $('#canvas').addEventListener('mousemove', (e) => {
        if (player != board.player) return;
        board.hover(e.offsetX, e.offsetY);
        board.repaint();
    });

    $('#canvas').addEventListener('click', (e) => {
        if (player != board.player) return;
        board.click(e.offsetX, e.offsetY);
        board.repaint();
        conn.send(JSON.stringify({Action: 'Move', Key: game.id, Payload: JSON.stringify(board.savePoints())}));
    });

    conn.onmessage = e => {
        const json = JSON.parse(e.data);
        console.log(json);
        if (json.Action == "New") {
            game = {id: json.Key};
            return;
        }
        if (json.Action == "Join" || json.Action == "Move") {
            const pts = JSON.parse(json.Payload);
            board.history.push(JSON.stringify(pts));
            board.loadPoints(pts);
            let black = 0;
            let white = 0;
            pts.forEach(p => {
                if (p.player == 'black') {
                    black++;    
                }
                if (p.player == 'white') {
                    white++;
                }
            });
            board.player = black > white ? 'white' : 'black';
            board.repaint();
            return;
        }

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
                console.log(opt.value);
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

    setInterval(e => {
        if (!conn.readyState == 1) return;
        conn.send(JSON.stringify({Action: 'List'}));
    }, 1000);
});
