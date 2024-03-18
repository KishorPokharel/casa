$(function () {
  const messageInput = $('.messageInput');
  const roomID = $('input[name=roomID]').val();
  const userIDString = $('input[name=userID]').val();
  const userID = parseInt(userIDString);
  const lastChild = $('.messages > :last-child')[0]
  lastChild && lastChild.scrollIntoView(false); // scroll to bottom

  let socket = null;
  window.socket = socket;

  if (!window['WebSocket']) {
    alert('browser not supported');
    return;
  } else {
    let protocol = window.location.protocol === 'http:' ? 'ws:' : 'wss:';
    let hostport = window.location.host;
    socket = new WebSocket(`${protocol}//${hostport}/ws/${roomID}`);
    socket.onclose = function () {
      alert('Connection closed by server');
    };
    socket.onmessage = function (e) {
      let msg = JSON.parse(e.data);
      let className;
      if (msg.sender_id === userID) {
        className = 'message message-my';
        $('.messages').append(
          $('<div>')
            .addClass(className)
            .append(
              $('<p>').text(msg.content),
              $('<small>').text(msg.created_at),
            ),
        );
      } else {
        className = 'message';
        $('.messages').append(
          $('<div>')
            .addClass(className)
            .append(
              $('<p>').text(msg.content),
              $('<small>').text(msg.created_at),
            ),
        );
      }
      const lastChild = $('.messages > :last-child')[0]
      lastChild && lastChild.scrollIntoView(false); // scroll to bottom
      // $('.messages > :last-child')[0].scrollIntoView(false); // scroll to bottom
    };
  }

  messageInput.on('keyup', e => {
    if (e.keyCode === 13) {
      if (!messageInput.val()) {
        return;
      }
      socket.send(JSON.stringify({content: messageInput.val()}));
      messageInput.val('');
    }
  });
});
