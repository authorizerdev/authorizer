'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-clickable.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-clickable.cjs.dev.js");
}
