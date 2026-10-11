// Remove credentials saved by older versions of the console.
sessionStorage.removeItem('taskflow-token');
const form = document.querySelector('#login-form');
const button = document.querySelector('#login-button');
const error = document.querySelector('#login-error');
form.addEventListener('submit', async (event) => {
  event.preventDefault();
  button.disabled = true;
  error.textContent = '';
  try {
    const values = new FormData(form);
    const response = await fetch('/auth/login', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: values.get('username'),
        password: values.get('password') }),
    });
    const result = await response.json();
    if (!response.ok) throw new Error(result.error || '登录失败');
    form.reset();
    location.replace('/');
  } catch (err) { error.textContent = err.message; }
  finally { button.disabled = false; }
});
