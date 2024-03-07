const locationInput = document.querySelector('#locationInput');
const locationResultsContainer = document.querySelector('.locationResultsContainer');
const locationResults = document.querySelector('#locationResults');
const locationBoxes = document.querySelectorAll('.location');
const latitudeInput = document.querySelector("#latitudeInput");
const longitudeInput = document.querySelector("#longitudeInput");

locationBoxes.forEach(box => {
  box.addEventListener("click", (e) => {
    const target = e.target.closest(".location");
    locationInput.value = target.dataset.text;
    locationResults.innerHTML = "";
  });
});

locationInput.addEventListener(
  'keyup',
  debounce(handleLocationInputChange, 1000),
);

function handleLocationInputChange(e) {
  e.preventDefault();
  let inputText = e.target.value;
  if (inputText === '') {
    return;
  }
  inputText = encodeURIComponent(inputText);
  fetchResults(inputText)
    .then(results => {
      if (results.features.length > 0) {
        let html = "";
        for (let result of results.features) {
          html += template(result.properties);
          console.log(result.properties);
        }
        locationResults.innerHTML = html;

        // listen for click on new elements
        const locationBoxes = document.querySelectorAll('.location');
        locationBoxes.forEach(box => {
          box.addEventListener("click", (e) => {
            const target = e.target;
            locationInput.value = target.dataset.text;
            latitudeInput.value = target.dataset.lat;
            longitudeInput.value = target.dataset.lon;
            locationResults.innerHTML = "";
          });
        });
      }
    })
    .catch(err => {
      console.error(err);
    });
}

function template(result) {
  return `
    <li class="location" data-text="${result.formatted}" data-lat="${result.lat}" data-lon="${result.lon}">
    ${result.formatted}
    </li>
  `
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

var requestOptions = {
  method: 'GET',
};

function fetchResults(text) {
  return fetch(
    `https://api.geoapify.com/v1/geocode/autocomplete?text=${text}&apiKey=d76967abd63741b894c6517816a7c8ec`,
    requestOptions,
  )
    .then(response => response.json())
    .then(result => {
      return result;
    })
    .catch(error => {
      throw new Error('Could not fetch results', {cause: error});
    });
}
