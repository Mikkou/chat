import { useEffect } from 'react';

export default function App() {
  useEffect(() => {
    const output = document.getElementById('output')!;
    const input = document.getElementById('input') as HTMLInputElement;
    const name = document.getElementById('name') as HTMLInputElement;
    let ws: WebSocket | null = null;

    const wsBase = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;

    const print = (message: string) => {
      const d = document.createElement('div');
      d.textContent = message;
      output.appendChild(d);
      output.scroll(0, output.scrollHeight);
    };

    const onOpen = (e: Event) => {
      e.preventDefault();
      if (ws) return false;
      if (!name.value.trim()) {
        alert('name is required');
        return false;
      }
      ws = new WebSocket(wsBase + '?name=' + encodeURIComponent(name.value));
      ws.onopen = () => print('OPEN');
      ws.onclose = () => {
        print('CLOSE');
        ws = null;
      };
      ws.onmessage = (evt) => print(evt.data);
      ws.onerror = () => print('ERROR');
      return false;
    };

    const onSend = (e: Event) => {
      e.preventDefault();
      if (!ws) return false;
      print(name.value + ': ' + input.value);
      ws.send(input.value);
      return false;
    };

    const onClose = (e: Event) => {
      e.preventDefault();
      if (!ws) return false;
      ws.close();
      return false;
    };

    const openBtn = document.getElementById('open')!;
    const sendBtn = document.getElementById('send')!;
    const closeBtn = document.getElementById('close')!;

    openBtn.addEventListener('click', onOpen);
    sendBtn.addEventListener('click', onSend);
    closeBtn.addEventListener('click', onClose);

    return () => {
      openBtn.removeEventListener('click', onOpen);
      sendBtn.removeEventListener('click', onSend);
      closeBtn.removeEventListener('click', onClose);
      ws?.close();
    };
  }, []);

  return (
      <table>
        <tbody>
        <tr>
          <td valign="top" width="50%">
            <p>
              Click &quot;Open&quot; to create a connection to the server,
              &quot;Send&quot; to send a message and &quot;Close&quot; to close the connection.
            </p>
            <form onSubmit={(e) => e.preventDefault()}>
              <button type="button" id="open">
                Open
              </button>
              <button type="button" id="close">
                Close
              </button>
              <input id="name" type="text" placeholder="your name" />
              <p>
                <input id="input" type="text" defaultValue="Hello world!" />
                <button type="button" id="send">
                  Send
                </button>
              </p>
            </form>
          </td>
          <td valign="top" width="50%">
            <div id="output" style={{ maxHeight: '70vh', overflowY: 'scroll' }} />
          </td>
        </tr>
        </tbody>
      </table>
  );
}