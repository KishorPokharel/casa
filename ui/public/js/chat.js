$(function () {
  const messageInput = $('.messageInput');
  const roomID = $("input[name=roomID]").val();
  alert(roomID);

  let socket = null;
  window.socket = socket;
  if (!window['WebSocket']) {
    alert('browser not supported');
    return;
  } else {
    let protocol = window.location.protocol === 'http:' ? 'ws:' : 'wss:';
    socket = new WebSocket(`${protocol}//localhost:3000/ws/${roomID}`);
    socket.onclose = function () {
      alert('Connection closed by server');
    };
    socket.onmessage = function (e) {
      let msg = JSON.parse(e.data);
    }
  }
});
