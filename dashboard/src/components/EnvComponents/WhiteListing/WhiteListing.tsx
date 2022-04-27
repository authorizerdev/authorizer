import React from "react";
import { Flex, Stack, Center, Text, useMediaQuery } from "@chakra-ui/react";
import InputField from "../../InputField";
import { ArrayInputType } from "../../../constants";
const WhiteListing = ({ variables, setVariables }: any) => {
  const [isNotSmallerScreen] = useMediaQuery("(min-width:600px)");
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
        White Listing
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Allowed Origins:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
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
