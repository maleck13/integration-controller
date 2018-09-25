/*
 * Copyright (C) 2016 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package io.syndesis.server.connector.generator.swagger.util;

import org.junit.Test;

import static io.syndesis.server.connector.generator.swagger.util.XmlSchemaHelper.XML_SCHEMA_PREFIX;

import static org.assertj.core.api.Assertions.assertThat;

public class XmlSchemaHelperTest {

    @Test
    public void shouldConvertJsonSchemaToXsdTypes() {
        assertThat(XmlSchemaHelper.toXsdType("boolean")).isEqualTo(XML_SCHEMA_PREFIX + ":boolean");
        assertThat(XmlSchemaHelper.toXsdType("number")).isEqualTo(XML_SCHEMA_PREFIX + ":decimal");
        assertThat(XmlSchemaHelper.toXsdType("string")).isEqualTo(XML_SCHEMA_PREFIX + ":string");
        assertThat(XmlSchemaHelper.toXsdType("integer")).isEqualTo(XML_SCHEMA_PREFIX + ":integer");
    }

}
