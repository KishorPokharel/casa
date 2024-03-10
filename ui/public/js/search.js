const searchInput = document.querySelector('#searchInput');
const locationResults = document.querySelector('#locationResults');

searchInput.addEventListener('keyup', debounce(showSuggestions), 800);

function showSuggestions(e) {
  const value = e.target.value.trim();
  if (value === '') {
    return;
  }
  const queryString = new URLSearchParams({query: value}).toString();
  fetch(`/locations?${queryString}`)
    .then(res => res.json())
    .then(data => {
      let html = "";
      for (let match of data.matches) {
        html += `<li class="location" data-text="${match.Target}">${match.Target}</li>`;
      }
      locationResults.innerHTML = html;

      // listen for click on new elements
      const locationBoxes = document.querySelectorAll('.location');
      locationBoxes.forEach(box => {
        box.addEventListener("click", (e) => {
          const target = e.target;
          searchInput.value = target.dataset.text;
          locationResults.innerHTML = "";
          searchInput.focus();
        });
      });
    })
    .catch(err => {
      console.error(err);
    });
}

function debounce(func, timeout = 300) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      func.apply(this, args);
    }, timeout);
  };
}
