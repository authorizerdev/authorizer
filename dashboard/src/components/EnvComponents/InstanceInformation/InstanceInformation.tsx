import React from "react";
import InputField from "../../InputField";
import { Flex, Stack, Center, Text, Box } from "@chakra-ui/react";
import {
  envVarTypes,
  TextInputType,
  HiddenInputType,
} from "../../../constants";

const InstantInformation = ({
  envVariables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
}: any) => {
  return (
    <Box>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Your instance information
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Client ID</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={envVariables}
              setVariables={() => {}}
              inputType={TextInputType.CLIENT_ID}
              placeholder="Client ID"
              readOnly
            />
          </Center>
        </Flex>
        <Flex>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Client Secret</Text>
          </Flex>
          <Center w="70%">
            <InputField
              variables={envVariables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.CLIENT_SECRET}
              placeholder="Client Secret"
              readOnly
            />
          </Center>
        </Flex>
      </Stack>
    </Box>
  );
};

export default InstantInformation;
