'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-image.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-image.cjs.dev.js");
}
