import React from "react";
import InputField from "../../InputField";
import {
  Flex,
  Stack,
  Center,
  Text,
  Box,
  useMediaQuery,
} from "@chakra-ui/react";
import { TextInputType, HiddenInputType } from "../../../constants";

const InstantInformation = ({
  envVariables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
}: any) => {
  const [isNotSmallerScreen] = useMediaQuery("(min-width:600px)");
  return (
    <div>
      <Box>
        {" "}
        <Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={6}>
          Your instance information
        </Text>
        <Stack spacing={6} padding="2% 0%">
          <Flex direction={isNotSmallerScreen ? "row" : "column"}>
            <Flex w="30%" justifyContent="start" alignItems="center">
              <Text fontSize="sm">Client ID</Text>
            </Flex>
            <Center
              w={isNotSmallerScreen ? "70%" : "100%"}
              mt={isNotSmallerScreen ? "0" : "3"}
            >
              <InputField
                variables={envVariables}
                setVariables={() => {}}
                inputType={TextInputType.CLIENT_ID}
                placeholder="Client ID"
                readOnly
              />
            </Center>
          </Flex>
          <Flex direction={isNotSmallerScreen ? "row" : "column"}>
            <Flex w="30%" justifyContent="start" alignItems="center">
              <Text fontSize="sm">Client Secret</Text>
            </Flex>
            <Center
              w={isNotSmallerScreen ? "70%" : "100%"}
              mt={isNotSmallerScreen ? "0" : "3"}
            >
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
    </div>
  );
};

export default InstantInformation;
