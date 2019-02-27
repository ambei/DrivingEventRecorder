import React, { Component } from "react";
import Axios from "../../utils/utils";
import backendConfig from "../../config/backendConfig";
import {
  Collapse,
  Radio,
  Checkbox,
  Button,
  Spin,
  Alert,
  notification,
  Input
} from "antd";
import dayjs from "dayjs";
import "./index.less";

export default class EventRecorder extends Component {
  state = {
    loaded: false,
    buttonLoading: false,
    event: { eventId: "-1", eventRecorder: [] }
  };

  componentDidMount = () => {
    Axios.ajax({
      url: backendConfig.eventsDefinitionApi,
      method: "GET"
    }).then(res => {
      this.setState({
        eventsDefinition: res.data,
        loaded: true
      });
    });
  };

  handleSubmit = () => {
    const time = dayjs().format("YYYY-MM-DDTHH:mm:ss");
    const event = this.state.event;
    const unusedRadios = event.eventRecorder.filter(
      value => value.type === "radio" && value.value === null
    );
    if (event.eventId === "-1") {
      notification.error({
        message: "提交错误",
        description: "请至少选择一项事件!"
      });
      return;
    }
    if (unusedRadios.length !== 0) {
      notification.error({
        message: "提交错误",
        description: "有单选项目未被选中!"
      });
      return;
    }

    this.setState({ buttonLoading: true });
    const eventString =
      event.eventId +
      "-" +
      event.eventRecorder
        .map(e =>
          e.type === "check"
            ? e.value === null
              ? ""
              : e.value.join("")
            : e.value
        )
        .join("");

    console.log(eventString);
    Axios.ajax({
      url: backendConfig.eventApi,
      method: "POST",
      data: { time, event: eventString }
    }).then(
      () => {
        this.setState({ buttonLoading: false });
      },
      () => {
        this.setState({ buttonLoading: false });
      }
    );
  };

  handleCodeChange = (eventGroup, value) => {
    const eventRecorder = this.state.event.eventRecorder.map(event => {
      return eventGroup === event.group
        ? { group: eventGroup, type: event.type, value }
        : event;
    });
    this.setState({
      event: { eventId: this.state.event.eventId, eventRecorder }
    });
  };

  handleEventChange = eventId => {
    if (eventId === undefined) {
      this.setState({ event: { eventId: "-1", eventRecorder: [] } });
      return;
    }
    const eventOptions = this.state.eventsDefinition[eventId - 1].option;
    const eventRecorder = eventOptions.map(option => {
      return {
        group: option.groupId,
        type: option.groupType,
        value: null
      };
    });
    const event = { eventId, eventRecorder };
    this.setState({ event });
  };

  renderEvents = () => {
    if (this.state.loaded) {
      return (
        <Collapse
          bordered={false}
          accordion
          className="events-wrap"
          onChange={this.handleEventChange}
          destroyInactivePanel
        >
          {this.state.eventsDefinition.map(event => {
            const { id, description, option } = event;
            return (
              <Collapse.Panel header={id + ". " + description} key={id}>
                {this.renderOption(option)}
              </Collapse.Panel>
            );
          })}
        </Collapse>
      );
    } else {
      return (
        <Spin tip="Loading..." className="events-wrap">
          <Alert
            message="正在加载"
            description="正在加载事件列表, 请稍候..."
            type="info"
            className="events-wrap"
          />
        </Spin>
      );
    }
  };

  renderOption = option => {
    return option.map(opt => {
      const { groupId, groupType, content } = opt;
      if (groupType === "radio") {
        return (
          <Radio.Group
            name={groupId}
            key={groupId}
            className="option-group"
            onChange={e => this.handleCodeChange(groupId, e.target.value)}
          >
            {content.map(content => {
              return (
                <Radio value={content.code} key={content.code}>
                  {content.description}
                </Radio>
              );
            })}
          </Radio.Group>
        );
      } else if (groupType === "check") {
        return (
          <Checkbox.Group
            name={groupId}
            key={groupId}
            className="option-group"
            onChange={checkedList =>
              this.handleCodeChange(groupId, checkedList.sort())
            }
          >
            {content.map(content => {
              return (
                <Checkbox value={content.code} key={content.code}>
                  {content.description}
                </Checkbox>
              );
            })}
          </Checkbox.Group>
        );
      } else if (groupType === "text") {
        //TODO: Text submission
        return content.map(content => {
          return (
            <Input
              key={content.code}
              placeholder={content.description}
              className="input"
            />
          );
        });
      }
      console.log(opt);
      return <span>Unkonwn option</span>;
    });
  };

  render() {
    return (
      <div>
        <Button
          type="primary"
          onClick={this.handleSubmit}
          loading={this.state.buttonLoading}
          className="submit-button"
        >
          {this.state.buttonLoading ? "提交中..." : "提交"}
        </Button>
        {this.renderEvents()}
      </div>
    );
  }
}
