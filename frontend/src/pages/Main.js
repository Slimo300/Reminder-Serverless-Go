import React, { useEffect, useState } from "react";
import { Navigate } from "react-router-dom";

import { ModalUserProfile } from "../modals/Profile";
import { getCurrentUser } from "../requests/Cognito";
import { ModalAlarmCreator } from "../modals/AlarmCreator";
import { getCronDescription } from "../crons";
import axiosObject from "../requests/API";

const Main = ({ profileShow, toggleProfile, loggedIn }) => {

    return (
        <div>
            {loggedIn?<AuthMain profileShow={profileShow} toggleProfile={toggleProfile}/>:<Navigate to="/login"/>}
        </div>
    );
}

const AuthMain = ({profileShow, toggleProfile}) => {

    const [createAlarmShow, setCreateAlarmShow] = useState(false);
    const toggleCreateAlarm = (e) => {
        setCreateAlarmShow(!createAlarmShow);
    }
    
    const [phoneNumber, setPhoneNumber] = useState("");

    useEffect(() => {
        getCurrentUser((err, attributes) => {
            if (err) {
                console.log(err);
                return;
            }
            for (let i = 0; i < attributes.length; i++) {
                if (attributes[i].Name === "phone_number") {
                    setPhoneNumber(attributes[i].Value);
                }
            }
        });
    }, []);

    useEffect(() => {
        const getAlarms = async () => { 
            const response = await axiosObject.get("/alarms");
            console.log(response.data);
            setEvents(response.data);
        }

        getAlarms();
    }, []);

    const [events, setEvents] = useState([]);
    
    const handleDelete = async (index) => {
        try {
            await axiosObject.delete(`/alarms/${events[index].EventID}`);
            const newEvents = events.filter((_, i) => i !== index);
            setEvents(newEvents)
        } catch(err) {
            alert(err.response.data);
        }
    };

    return (
        <div className="container" >
            <button className="btn btn-primary mt-3 ml-3" onClick={toggleCreateAlarm}>New Alarm</button>
            <div className="container mt-4">
            <table className="table table-bordered">
                <thead>
                <tr>
                    <th>Name</th>
                    <th>Events</th>
                    <th>Delete</th>
                </tr>
                </thead>
                <tbody>
                {events?events.map((item, index) => (
                    <tr key={index}>
                        <td>{item.Title}</td>
                        <td>
                            {Object.keys(item.Dates) && Object.keys(item.Dates).length !== 0?<div>
                                <h5>One Time Events:</h5>
                                <ul>
                                    {Object.values(item.Dates).map((event, index) => {
                                        return <li key={index}>{new Date(event).toLocaleString()}</li>
                                    })}
                                </ul>
                            </div>:null}
                            {Object.keys(item.Crons) && Object.keys(item.Crons).length !== 0?<div>
                                <h5>Cron Events:</h5>
                                <ul>
                                    {Object.values(item.Crons).map((cron, index) => {
                                        return <li key={index}>{getCronDescription(cron)}</li>
                                    })}
                                </ul>
                            </div>:null}
                        </td>
                        <td>
                            <button className="btn btn-danger" onClick={() => handleDelete(index)}> Delete </button>
                        </td>
                    </tr>
                )):null}
                </tbody>
            </table>
            </div>
            <ModalAlarmCreator show={createAlarmShow} toggle={toggleCreateAlarm} setEvents={setEvents}/>
            <ModalUserProfile show={profileShow} toggle={toggleProfile} phoneNumber={phoneNumber} setPhoneNumber={setPhoneNumber} />
        </div>
    )
}

export default Main;