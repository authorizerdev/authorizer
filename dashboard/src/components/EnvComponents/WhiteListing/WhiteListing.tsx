import React from "react";
import {
  Box,
  Divider,
  Flex,
  Stack,
  Center,
  Text,
  Button,
  Input,
  InputGroup,
  InputRightElement,
  useToast,
} from "@chakra-ui/react";
import InputField from "../../InputField";
import { ArrayInputType } from "../../../constants";
const WhiteListing = ({ variables, setVariables }: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        White Listing
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Allowed Origins:</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={ArrayInputType.ALLOWED_ORIGINS}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default WhiteListing;
