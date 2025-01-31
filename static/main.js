// 認証状態をチェックしてUIを更新する関数
async function updateAuthUI(user = null) {
    const authSection = document.getElementById('authSection');
    const contentSection = document.getElementById('contentSection');
    const usernameSpan = document.getElementById('username');

    if (user) {
        // デバッグ用にユーザー情報を表示
        console.log('Received user:', user);

        // ユーザー情報を保存
        usernameSpan.textContent = user.username;
        usernameSpan.dataset.userId = String(user.id);

        // UIを更新
        authSection.style.display = 'none';
        contentSection.style.display = 'block';

        // メッセージ一覧を取得
        try {
            await loadMessages();
        } catch (error) {
            console.error('メッセージ一覧の取得に失敗しました:', error);
        }
    } else {
        authSection.style.display = 'block';
        contentSection.style.display = 'none';
        usernameSpan.textContent = '';
        delete usernameSpan.dataset.userId;
    }
}

// ユーザー登録関数
async function register() {
    const username = document.getElementById('registerUsername').value;
    const password = document.getElementById('registerPassword').value;

    try {
        const response = await fetch('/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) {
            const user = await response.json();
            alert('ユーザー登録が完了しました。ログインしてください。');
            showLoginForm();
        } else {
            const error = await response.text();
            alert('登録に失敗しました: ' + error);
        }
    } catch (error) {
        console.error('エラー:', error);
        alert('登録中にエラーが発生しました');
    }
}

// ログイン関数
async function login() {
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;

    try {
        const response = await fetch('/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) {
            const user = await response.json();
            console.log('Login successful:', user); // デバッグ用
            await updateAuthUI(user);
        } else {
            const error = await response.text();
            alert('ログインに失敗しました: ' + error);
        }
    } catch (error) {
        console.error('ログインエラー:', error);
        alert('ログイン中にエラーが発生しました');
    }
}

// ログアウト関数
async function logout() {
    try {
        const response = await fetch('/logout', {
            method: 'POST'
        });

        if (response.ok) {
            updateAuthUI(null);
        }
    } catch (error) {
        console.error('エラー:', error);
        alert('ログアウト中にエラーが発生しました');
    }
}

// 登録フォームを表示
function showRegisterForm() {
    document.getElementById('loginForm').style.display = 'none';
    document.getElementById('registerForm').style.display = 'block';
}

// ログインフォームを表示
function showLoginForm() {
    document.getElementById('loginForm').style.display = 'block';
    document.getElementById('registerForm').style.display = 'none';
}

// メッセージを表示する関数
function displayMessage(message, prepend = false) {
    const messageList = document.getElementById('messageList');
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message';
    messageDiv.id = `message-${message.id}`;
    
    const time = new Date(message.timestamp).toLocaleString('ja-JP');
    const currentUserID = getCurrentUserID();
    const isOwner = currentUserID === message.user_id;
    
    messageDiv.innerHTML = `
        <div class="message-header">
            <div class="message-info">
                <p class="message-text" id="message-text-${message.id}">${escapeHtml(message.text)}</p>
                <small class="message-author">投稿者: ${escapeHtml(message.username)}</small>
            </div>
            <div class="message-actions">
                <button class="reply-button" onclick="startReply(${message.id})">返信</button>
                ${isOwner ? `
                    <button class="edit-button" onclick="startEdit(${message.id}, '${escapeHtml(message.text)}')">編集</button>
                    <button class="delete-button" onclick="deleteMessage(${message.id})">削除</button>
                ` : ''}
            </div>
        </div>
        <div class="message-time">${time}</div>
        <div class="edit-form" id="edit-form-${message.id}" style="display: none;">
            <input type="text" id="edit-input-${message.id}" class="edit-input" value="${escapeHtml(message.text)}">
            <div class="edit-buttons">
                <button class="save-button" onclick="saveEdit(${message.id})">保存</button>
                <button class="cancel-button" onclick="cancelEdit(${message.id})">キャンセル</button>
            </div>
        </div>
        <div class="reply-form" id="reply-form-${message.id}" style="display: none;">
            <input type="text" id="reply-input-${message.id}" class="reply-input" placeholder="返信を入力...">
            <div class="reply-buttons">
                <button class="send-reply-button" onclick="sendReply(${message.id})">送信</button>
                <button class="cancel-button" onclick="cancelReply(${message.id})">キャンセル</button>
            </div>
        </div>
        ${message.replies && message.replies.length > 0 ? `
            <div class="replies" id="replies-${message.id}">
                ${message.replies.map(reply => `
                    <div class="reply" id="message-${reply.id}">
                        <div class="reply-header">
                            <p class="reply-text" id="message-text-${reply.id}">${escapeHtml(reply.text)}</p>
                            ${currentUserID === reply.user_id ? `
                                <div class="message-actions">
                                    <button class="edit-button" onclick="startEdit(${reply.id}, '${escapeHtml(reply.text)}')">編集</button>
                                    <button class="delete-button" onclick="deleteMessage(${reply.id})">削除</button>
                                </div>
                            ` : ''}
                        </div>
                        <div class="reply-meta">
                            <span class="message-username">${escapeHtml(reply.username)}</span>
                            <span class="message-time">${new Date(reply.timestamp).toLocaleString('ja-JP')}</span>
                        </div>
                        <div class="edit-form" id="edit-form-${reply.id}" style="display: none;">
                            <input type="text" id="edit-input-${reply.id}" class="edit-input" value="${escapeHtml(reply.text)}">
                            <div class="edit-buttons">
                                <button class="save-button" onclick="saveEdit(${reply.id})">保存</button>
                                <button class="cancel-button" onclick="cancelEdit(${reply.id})">キャンセル</button>
                            </div>
                        </div>
                    </div>
                `).join('')}
            </div>
        ` : ''}
    `;

    // メッセージを追加する位置を決定
    if (prepend) {
        messageList.insertBefore(messageDiv, messageList.firstChild);
        messageDiv.style.opacity = '0';
        setTimeout(() => {
            messageDiv.style.opacity = '1';
        }, 10);
    } else {
        messageList.appendChild(messageDiv);
    }
}

// メッセージを送信する関数
async function sendMessage() {
    const input = document.getElementById('messageInput');
    const text = input.value.trim();
    
    if (!text) return;

    try {
        const response = await fetch('/messages/save', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ text: text })
        });

        if (response.ok) {
            const message = await response.json();
            input.value = '';
            displayMessage(message, true); // 新しいメッセージを先頭に追加
        } else {
            const error = await response.text();
            console.error('メッセージの送信に失敗しました:', error);
            if (response.status === 401) {
                updateAuthUI(null);
            }
        }
    } catch (error) {
        console.error('エラー:', error);
        alert('メッセージの送信中にエラーが発生しました');
    }
}

// メッセージを削除する関数
async function deleteMessage(id) {
    if (!confirm('このメッセージを削除しますか？')) {
        return;
    }

    try {
        const response = await fetch(`/messages/delete?id=${id}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            const messageDiv = document.getElementById(`message-${id}`);
            if (messageDiv) {
                // メッセージをフェードアウトして削除
                messageDiv.style.opacity = '0';
                setTimeout(() => {
                    messageDiv.remove();
                }, 300); // transitionの時間と同じにする
            }
        } else {
            const error = await response.text();
            console.error('メッセージの削除に失敗しました:', error);
            if (response.status === 401) {
                updateAuthUI(null);
            } else {
                alert('メッセージの削除に失敗しました: ' + error);
            }
        }
    } catch (error) {
        console.error('エラー:', error);
        alert('メッセージの削除中にエラーが発生しました');
    }
}

// メッセージ一覧を取得して表示する関数
async function loadMessages() {
    try {
        const response = await fetch('/messages');
        if (response.status === 401) {
            updateAuthUI(null);
            return;
        }

        const messages = await response.json();
        const messageList = document.getElementById('messageList');
        messageList.innerHTML = '';

        // メッセージを表示
        messages.forEach(message => displayMessage(message));
    } catch (error) {
        console.error('エラー:', error);
        if (error.name === 'SyntaxError') {
            updateAuthUI(null);
        }
    }
}

// 現在のユーザーIDを取得する関数
function getCurrentUserID() {
    const usernameSpan = document.getElementById('username');
    const userId = usernameSpan.dataset.userId;
    console.log('Raw user ID:', userId); // デバッグ用
    if (!userId) return null;
    const parsedId = parseInt(userId, 10);
    console.log('Parsed user ID:', parsedId); // デバッグ用
    return parsedId;
}

// 編集モードを開始する関数
function startEdit(id, text) {
    const messageText = document.getElementById(`message-text-${id}`);
    const editForm = document.getElementById(`edit-form-${id}`);
    const editInput = document.getElementById(`edit-input-${id}`);

    messageText.style.opacity = '0';
    editInput.value = text;

    setTimeout(() => {
        messageText.style.display = 'none';
        editForm.style.display = 'block';
        editForm.classList.add('active');
        editInput.focus();
    }, 300);
}

// 編集をキャンセルする関数
function cancelEdit(id) {
    const messageText = document.getElementById(`message-text-${id}`);
    const editForm = document.getElementById(`edit-form-${id}`);

    editForm.classList.remove('active');
    setTimeout(() => {
        editForm.style.display = 'none';
        messageText.style.display = 'block';
        messageText.style.opacity = '1';
    }, 300);
}

// 編集を保存する関数
async function saveEdit(id) {
    const editInput = document.getElementById(`edit-input-${id}`);
    const text = editInput.value.trim();

    if (!text) return;

    try {
        const response = await fetch(`/messages/update?id=${id}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ text: text })
        });

        if (response.ok) {
            const messageText = document.getElementById(`message-text-${id}`);
            const editForm = document.getElementById(`edit-form-${id}`);
            
            editForm.classList.remove('active');
            setTimeout(() => {
                editForm.style.display = 'none';
                messageText.style.display = 'block';
                messageText.textContent = text;
                messageText.style.opacity = '0';
                setTimeout(() => {
                    messageText.style.opacity = '1';
                }, 10);
            }, 300);
        } else {
            const error = await response.text();
            console.error('メッセージの更新に失敗しました:', error);
            alert('メッセージの更新に失敗しました');
        }
    } catch (error) {
        console.error('エラー:', error);
        alert('メッセージの更新中にエラーが発生しました');
    }
}

// HTMLをエスケープする関数
function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// 返信モードを開始する関数
function startReply(id) {
    const replyForm = document.getElementById(`reply-form-${id}`);
    replyForm.style.display = 'block';
    replyForm.classList.add('active');
    document.getElementById(`reply-input-${id}`).focus();
}

// 返信をキャンセルする関数
function cancelReply(id) {
    const replyForm = document.getElementById(`reply-form-${id}`);
    replyForm.classList.remove('active');
    setTimeout(() => {
        replyForm.style.display = 'none';
        document.getElementById(`reply-input-${id}`).value = '';
    }, 300);
}

// 返信を送信する関数
async function sendReply(id) {
    const input = document.getElementById(`reply-input-${id}`);
    const text = input.value.trim();
    
    if (!text) return;

    try {
        const response = await fetch('/messages/save', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                text: text,
                reply_to: parseInt(id, 10)
            })
        });

        if (response.ok) {
            const message = await response.json();
            // 返信を表示
            const repliesDiv = document.getElementById(`replies-${id}`);
            if (!repliesDiv) {
                const newRepliesDiv = document.createElement('div');
                newRepliesDiv.className = 'replies';
                newRepliesDiv.id = `replies-${id}`;
                document.getElementById(`message-${id}`).appendChild(newRepliesDiv);
            }
            
            const replyDiv = document.createElement('div');
            replyDiv.className = 'reply';
            replyDiv.id = `message-${message.id}`;
            replyDiv.style.opacity = '0';
            
            const currentUserID = getCurrentUserID();
            replyDiv.innerHTML = `
                <div class="reply-header">
                    <p class="reply-text" id="message-text-${message.id}">${escapeHtml(message.text)}</p>
                    ${currentUserID === message.user_id ? `
                        <div class="message-actions">
                            <button class="edit-button" onclick="startEdit(${message.id}, '${escapeHtml(message.text)}')">編集</button>
                            <button class="delete-button" onclick="deleteMessage(${message.id})">削除</button>
                        </div>
                    ` : ''}
                </div>
                <div class="reply-meta">
                    <span class="message-username">${escapeHtml(message.username)}</span>
                    <span class="message-time">${new Date(message.timestamp).toLocaleString('ja-JP')}</span>
                </div>
                <div class="edit-form" id="edit-form-${message.id}" style="display: none;">
                    <input type="text" id="edit-input-${message.id}" class="edit-input" value="${escapeHtml(message.text)}">
                    <div class="edit-buttons">
                        <button class="save-button" onclick="saveEdit(${message.id})">保存</button>
                        <button class="cancel-button" onclick="cancelEdit(${message.id})">キャンセル</button>
                    </div>
                </div>
            `;
            
            (repliesDiv || newRepliesDiv).appendChild(replyDiv);
            setTimeout(() => {
                replyDiv.style.opacity = '1';
            }, 10);
            
            input.value = '';
            cancelReply(id);
        } else {
            const error = await response.text();
            console.error('返信の送信に失敗しました:', error);
            alert('返信の送信に失敗しました');
        }
    } catch (error) {
        console.error('エラー:', error);
        alert('返信の送信中にエラーが発生しました');
    }
}

// Enterキーでメッセージを送信
document.getElementById('messageInput').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        sendMessage();
    }
});

// 返信フォームでEnterキーを押したときに返信を送信
document.addEventListener('keypress', function(e) {
    if (e.key === 'Enter' && e.target.classList.contains('reply-input')) {
        const messageId = e.target.id.replace('reply-input-', '');
        sendReply(messageId);
    }
});

// 初期ロード
loadMessages();

// 定期的に更新（5秒ごと）
setInterval(loadMessages, 5000);
