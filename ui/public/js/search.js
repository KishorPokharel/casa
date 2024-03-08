const searchInput = document.querySelector('#searchInput');

searchInput.addEventListener('keyup', showSuggestions);

var locations;
var fuse;

fetch('/locations')
  .then(res => res.json())
  .then(data => {
    // locations = data.locations;
    fuse = new Fuse(data.locations, {includeScore: true});
    console.log(data);
  })
  .catch(err => {
    console.error(err);
  });

function showSuggestions(e) {
  const value = e.target.value;
  console.log('Locations: ', locations);
  const results = fuse.search(value);
  console.log(results);
}

