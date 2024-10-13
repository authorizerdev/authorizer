'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-react-env.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-react-env.cjs.dev.js");
}
